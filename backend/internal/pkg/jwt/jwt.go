package jwt

import (
	"errors"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"github.com/google/uuid"
)

// Claims carried by the 15-minute access token. Stateless — middleware never
// hits the DB to authenticate.
type Claims struct {
	UserID       uuid.UUID `json:"uid"`
	Role         string    `json:"role"`
	DepartmentID uuid.UUID `json:"dep,omitempty"`
	Name         string    `json:"name"`
	jwt.RegisteredClaims
}

var ErrInvalid = errors.New("invalid token")

func Sign(secret string, userID uuid.UUID, role string, deptID uuid.UUID, name string, ttl time.Duration) (string, error) {
	now := time.Now()
	claims := Claims{
		UserID:       userID,
		Role:         role,
		DepartmentID: deptID,
		Name:         name,
		RegisteredClaims: jwt.RegisteredClaims{
			Subject:   userID.String(),
			IssuedAt:  jwt.NewNumericDate(now),
			ExpiresAt: jwt.NewNumericDate(now.Add(ttl)),
			Issuer:    "reimburseflow",
		},
	}
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	return token.SignedString([]byte(secret))
}

func Verify(secret, tokenStr string) (*Claims, error) {
	parsed, err := jwt.ParseWithClaims(tokenStr, &Claims{}, func(t *jwt.Token) (interface{}, error) {
		if _, ok := t.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, ErrInvalid
		}
		return []byte(secret), nil
	})
	if err != nil || !parsed.Valid {
		return nil, ErrInvalid
	}
	claims, ok := parsed.Claims.(*Claims)
	if !ok {
		return nil, ErrInvalid
	}
	return claims, nil
}
