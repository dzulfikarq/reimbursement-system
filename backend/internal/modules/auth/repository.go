package auth

import (
	"context"
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"time"

	"github.com/google/uuid"
	goredis "github.com/redis/go-redis/v9"
	"gorm.io/gorm"
)

// User maps the users table (auth-relevant subset).
type User struct {
	ID            uuid.UUID `gorm:"type:uuid;primaryKey"`
	DepartmentID  *uuid.UUID
	Name          string
	Email         string
	PasswordHash  string
	Role          string
	IsActive      bool
	CreatedAt     time.Time
	UpdatedAt     time.Time
}

func (User) TableName() string { return "users" }

type departmentRow struct {
	ID   uuid.UUID
	Name string
}

type Repository struct {
	db *gorm.DB
}

func NewRepository(db *gorm.DB) *Repository {
	return &Repository{db: db}
}

var ErrInvalidCredentials = errors.New("invalid credentials")
var ErrInactive = errors.New("account disabled")

func (r *Repository) FindByEmail(ctx context.Context, email string) (*User, error) {
	var u User
	err := r.db.WithContext(ctx).
		Select("users.id", "users.department_id", "users.name", "users.email",
			"users.password_hash", "users.role", "users.is_active", "users.created_at", "users.updated_at").
		Where("email = ?", email).
		First(&u).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, ErrInvalidCredentials
	}
	return &u, err
}

func (r *Repository) GetByID(ctx context.Context, id uuid.UUID) (*User, error) {
	var u User
	err := r.db.WithContext(ctx).
		Select("users.id", "users.department_id", "users.name", "users.email",
			"users.password_hash", "users.role", "users.is_active", "users.created_at", "users.updated_at").
		Where("users.id = ?", id).
		First(&u).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, ErrInvalidCredentials
	}
	return &u, err
}

func (r *Repository) DepartmentName(ctx context.Context, id uuid.UUID) (string, error) {
	var d departmentRow
	err := r.db.WithContext(ctx).Table("departments").
		Select("id", "name").Where("id = ?", id).First(&d).Error
	if err != nil {
		return "", err
	}
	return d.Name, nil
}

// --- refresh-token session store (Redis) ---
//
// Token wire format: "<familyID>.<jti>.<secret>". Redis holds sha256(secret)
// per jti plus the set of live jtis per family so that presenting a rotated-out
// token revokes the whole family (reuse detection, docs/06).

const (
	keyJTI  = "refresh:jti:" // + jti -> json{uid, hash}
	keyFAM  = "refresh:fam:" // + famid -> set of jtis
)

type sessionEntry struct {
	UserID uuid.UUID `json:"uid"`
	Hash   string    `json:"hash"`
}

type SessionStore struct {
	rdb *goredis.Client
	ttl time.Duration
}

func NewSessionStore(rdb *goredis.Client, ttl time.Duration) *SessionStore {
	return &SessionStore{rdb: rdb, ttl: ttl}
}

func randomHex(n int) string {
	b := make([]byte, n)
	_, _ = rand.Read(b)
	return hex.EncodeToString(b)
}

func hashSecret(secret string) string {
	sum := sha256.Sum256([]byte(secret))
	return hex.EncodeToString(sum[:])
}

// Issue creates a fresh family (login) and returns the opaque refresh token.
func (s *SessionStore) Issue(ctx context.Context, userID uuid.UUID) (string, error) {
	fam, jti, secret := randomHex(8), randomHex(8), randomHex(32)
	entry, _ := json.Marshal(sessionEntry{UserID: userID, Hash: hashSecret(secret)})
	pipe := s.rdb.TxPipeline()
	pipe.Set(ctx, keyJTI+jti, entry, s.ttl)
	pipe.SAdd(ctx, keyFAM+fam, jti)
	pipe.Expire(ctx, keyFAM+fam, s.ttl)
	if _, err := pipe.Exec(ctx); err != nil {
		return "", err
	}
	return fam + "." + jti + "." + secret, nil
}

// Rotate atomically consumes one jti and issues a sibling within the family.
// Returns ErrUnknownToken when the jti is gone (expired OR already used —
// reuse signal); caller then revokes the family.
func (s *SessionStore) Rotate(ctx context.Context, fam, jti, secret string) (string, uuid.UUID, error) {
	raw, err := s.rdb.GetDel(ctx, keyJTI+jti).Result()
	if err != nil {
		return "", uuid.Nil, ErrUnknownToken
	}
	var entry sessionEntry
	if err := json.Unmarshal([]byte(raw), &entry); err != nil ||
		entry.Hash != hashSecret(secret) {
		s.RevokeFamily(ctx, fam)
		return "", uuid.Nil, ErrUnknownToken
	}

	newJTI, newSecret := randomHex(8), randomHex(32)
	newRaw, _ := json.Marshal(sessionEntry{UserID: entry.UserID, Hash: hashSecret(newSecret)})
	pipe := s.rdb.TxPipeline()
	pipe.Set(ctx, keyJTI+newJTI, newRaw, s.ttl)
	pipe.SRem(ctx, keyFAM+fam, jti)
	pipe.SAdd(ctx, keyFAM+fam, newJTI)
	pipe.Expire(ctx, keyFAM+fam, s.ttl)
	if _, err := pipe.Exec(ctx); err != nil {
		return "", uuid.Nil, err
	}
	return fam + "." + newJTI + "." + newSecret, entry.UserID, nil
}

// RevokeFamily kills every live token in a chain — used on reuse detection and
// logout (a chain belongs to exactly one browser session).
func (s *SessionStore) RevokeFamily(ctx context.Context, fam string) error {
	jtis, _ := s.rdb.SMembers(ctx, keyFAM+fam).Result()
	pipe := s.rdb.TxPipeline()
	for _, j := range jtis {
		pipe.Del(ctx, keyJTI+j)
	}
	pipe.Del(ctx, keyFAM+fam)
	_, err := pipe.Exec(ctx)
	return err
}

var ErrUnknownToken = errors.New("unknown refresh token")
