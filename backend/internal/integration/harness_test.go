// Package integration runs the real router against real Postgres/Redis
// (docker compose services). Skipped when SKIP_INTEGRATION_TESTS=1.
package integration

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"net/http/cookiejar"
	"net/http/httptest"
	"net/url"
	"os"
	"strings"
	"sync"
	"testing"
	"time"

	goredis "github.com/redis/go-redis/v9"
	"gorm.io/gorm"

	"github.com/mumtaz/reimbursement-system/backend/internal/config"
	"github.com/mumtaz/reimbursement-system/backend/internal/database"
	"github.com/mumtaz/reimbursement-system/backend/internal/pkg/password"
	"github.com/mumtaz/reimbursement-system/backend/internal/server"
)

const (
	testPassword = "TestPass#123"
	appSecret    = "integration-test-secret"
)

var (
	hOnce   sync.Once
	hDB     *gorm.DB
	hRDB    *goredis.Client
	hServer *httptest.Server
	hURL    string
	hErr    error
)

func TestMain(m *testing.M) {
	if os.Getenv("SKIP_INTEGRATION_TESTS") == "1" {
		fmt.Println("skipping integration tests")
		os.Exit(0)
	}
	code := m.Run()
	if hServer != nil {
		hServer.Close()
	}
	if hRDB != nil {
		_ = hRDB.Close()
	}
	os.Exit(code)
}

func setup(t *testing.T) {
	t.Helper()
	hOnce.Do(startStack)
	if hErr != nil {
		t.Fatalf("integration stack failed to start: %v", hErr)
	}
}

func startStack() {
	logger := slog.New(slog.NewTextHandler(io.Discard, nil))

	host := getenv("TEST_PG_HOST", "localhost")
	port := getenv("TEST_PG_PORT", "5432")
	dsn := fmt.Sprintf("postgres://%s:%s@%s:%s/%s?sslmode=disable",
		getenv("TEST_PG_USER", "reimbursement"),
		getenv("TEST_PG_PASSWORD", "reimbursement"),
		host, port,
		getenv("TEST_PG_NAME", "reimbursement"),
	)
	if err := database.MigrateUp(dsn); err != nil {
		hErr = fmt.Errorf("migrate: %w", err)
		return
	}

	cfg := &config.Config{
		Env:                 "test",
		Port:                "0",
		DBHost:              host,
		DBPort:              port,
		DBUser:              getenv("TEST_PG_USER", "reimbursement"),
		DBPassword:          getenv("TEST_PG_PASSWORD", "reimbursement"),
		DBName:              getenv("TEST_PG_NAME", "reimbursement"),
		DBSSLMode:           "disable",
		RedisAddr:           getenv("TEST_REDIS_ADDR", "localhost:6379"),
		AppSecret:           appSecret,
		FrontendURL:         "http://localhost:5173",
		AccessTTL:           15 * time.Minute,
		RefreshTTL:          7 * 24 * time.Hour,
		CookieSecure:        false,
		MinioEndpoint:       "unused:9000",
		MinioBucket:         "attachments",
		MinioPublicEndpoint: "unused:9000",
		ApprovalT1:          "500000",
		ApprovalT2:          "5000000",
		ReceiptThreshold:    "500000",
	}

	var err error
	hDB, err = database.Connect(cfg, logger)
	if err != nil {
		hErr = fmt.Errorf("connect pg: %w", err)
		return
	}
	hRDB = goredis.NewClient(&goredis.Options{Addr: cfg.RedisAddr})
	hServer = httptest.NewServer(server.New(cfg, hDB, hRDB, nil))
	hURL = hServer.URL
}

func getenv(k, fallback string) string {
	if v := os.Getenv(k); v != "" {
		return v
	}
	return fallback
}

// --- seeding ---

func seedUser(t *testing.T, name, role string) (id, email string) {
	t.Helper()
	hash, err := password.Hash(testPassword)
	if err != nil {
		t.Fatal(err)
	}
	email = fmt.Sprintf("%s-%d@test.local", strings.ReplaceAll(strings.ToLower(name), " ", "."), time.Now().UnixNano())
	err = hDB.Raw(
		`INSERT INTO users (name, email, password_hash, role) VALUES (?, ?, ?, ?::user_role) RETURNING id::text`,
		name, email, hash, role).Scan(&id).Error
	if err != nil {
		t.Fatalf("seed user: %v", err)
	}
	return id, email
}

func seedCategory(t *testing.T, limit string) string {
	t.Helper()
	code := fmt.Sprintf("IT%d", time.Now().UnixNano()%1_000_000)
	var id string
	q := `INSERT INTO categories (code, name, monthly_limit_per_employee) VALUES (?, ?, ?::numeric) RETURNING id::text`
	args := []any{code, "Category " + code}
	if limit == "" {
		q = `INSERT INTO categories (code, name) VALUES (?, ?) RETURNING id::text`
	} else {
		args = append(args, limit)
	}
	if err := hDB.Raw(q, args...).Scan(&id).Error; err != nil {
		t.Fatalf("seed category: %v", err)
	}
	return id
}

// --- session helper ---

type session struct {
	t      *testing.T
	client *http.Client
	jar    *cookiejar.Jar
}

func login(t *testing.T, email, password string) *session {
	t.Helper()
	jar, err := cookiejar.New(nil)
	if err != nil {
		t.Fatal(err)
	}
	c := &http.Client{Jar: jar, CheckRedirect: func(*http.Request, []*http.Request) error { return http.ErrUseLastResponse }}

	body, _ := json.Marshal(map[string]string{"email": email, "password": password})
	resp := rawPost(t, c, hURL+"/api/v1/auth/login", body, "")
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		b, _ := io.ReadAll(resp.Body)
		t.Fatalf("login %s: got %d: %s", email, resp.StatusCode, b)
	}
	return &session{t: t, client: c, jar: jar}
}

// csrf reads the CURRENT csrf cookie — refresh/login may rotate it.
func (s *session) csrf() string {
	for _, ck := range s.jar.Cookies(mustParse(hURL)) {
		if ck.Name == "csrf_token" {
			return ck.Value
		}
	}
	return ""
}

func (s *session) do(method, path string, body any, _ bool) (int, map[string]any) {
	s.t.Helper()
	var rdr io.Reader = bytes.NewReader([]byte("{}"))
	if body != nil {
		b, err := json.Marshal(body)
		if err != nil {
			s.t.Fatal(err)
		}
		rdr = bytes.NewReader(b)
	}
	req, err := http.NewRequest(method, s.abs(path), rdr)
	if err != nil {
		s.t.Fatal(err)
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("X-CSRF-Token", s.csrf())
	resp, err := s.client.Do(req)
	if err != nil {
		s.t.Fatal(err)
	}
	defer resp.Body.Close()
	var out map[string]any
	raw, _ := io.ReadAll(resp.Body)
	_ = json.Unmarshal(raw, &out)
	return resp.StatusCode, out
}

func (s *session) abs(path string) string {
	if strings.HasPrefix(path, "http") {
		return path
	}
	return hURL + path
}

func rawPost(t *testing.T, c *http.Client, u string, body []byte, csrf string) *http.Response {
	t.Helper()
	req, err := http.NewRequest(http.MethodPost, u, bytes.NewReader(body))
	if err != nil {
		t.Fatal(err)
	}
	req.Header.Set("Content-Type", "application/json")
	if csrf != "" {
		req.Header.Set("X-CSRF-Token", csrf)
	}
	resp, err := c.Do(req)
	if err != nil {
		t.Fatal(err)
	}
	return resp
}

func mustParse(raw string) *url.URL {
	u, err := url.Parse(raw)
	if err != nil {
		panic(err)
	}
	return u
}
