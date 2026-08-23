package integration

import (
	"encoding/json"
	"net/http"
	"net/http/cookiejar"
	"testing"

	"github.com/mumtaz/reimbursement-system/backend/internal/pkg/password"
)

// Login with wrong password → 401, no cookies.
func TestAuthLoginRejectsBadCredentials(t *testing.T) {
	setup(t)
	jar, _ := cookiejar.New(nil)
	c := &http.Client{Jar: jar}

	body, _ := json.Marshal(map[string]string{"email": "nobody@test.local", "password": "wrong-password-long"})
	resp := rawPost(t, c, hURL+"/api/v1/auth/login", body, "")
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusUnauthorized {
		t.Fatalf("want 401, got %d", resp.StatusCode)
	}
	for _, ck := range jar.Cookies(mustParse(hURL)) {
		if ck.Name == "access_token" || ck.Name == "refresh_token" {
			t.Fatalf("session cookie %s must not be set on failed login", ck.Name)
		}
	}
}

// Full session lifecycle: login → me → refresh rotation → old refresh dead → logout.
func TestAuthSessionLifecycle(t *testing.T) {
	setup(t)
	_, email := seedUser(t, "Auth Flow", "employee")
	s := login(t, email, testPassword)

	// /me returns the user.
	code, out := s.do(http.MethodGet, "/api/v1/auth/me", nil, false)
	if code != http.StatusOK {
		t.Fatalf("/me: want 200, got %d", code)
	}
	user := out["data"].(map[string]any)["user"].(map[string]any)
	if user["email"] != email {
		t.Fatalf("/me email mismatch: %v", user["email"])
	}

	// /me without a session → 401.
	fresh, _ := cookiejar.New(nil)
	anon := &http.Client{Jar: fresh}
	resp, err := anon.Get(hURL + "/api/v1/auth/me")
	if err != nil {
		t.Fatal(err)
	}
	resp.Body.Close()
	if resp.StatusCode != http.StatusUnauthorized {
		t.Fatalf("anonymous /me: want 401, got %d", resp.StatusCode)
	}
}

func TestRefreshRotatesAndRevokesOldToken(t *testing.T) {
	setup(t)
	_, email := seedUser(t, "Refresher", "employee")
	s := login(t, email, testPassword)

	// First refresh succeeds and sets new cookies (rotation).
	code, _ := s.do(http.MethodPost, "/api/v1/auth/refresh", nil, false)
	if code != http.StatusOK {
		t.Fatalf("first refresh: want 200, got %d", code)
	}

	// Second refresh also succeeds — it uses the rotated-in token.
	code, _ = s.do(http.MethodPost, "/api/v1/auth/refresh", nil, false)
	if code != http.StatusOK {
		t.Fatalf("second refresh: want 200, got %d", code)
	}

	// Logout revokes the chain…
	code, _ = s.do(http.MethodPost, "/api/v1/auth/logout", nil, true)
	if code != http.StatusNoContent && code != http.StatusOK {
		t.Fatalf("logout: want 2xx, got %d", code)
	}

	// …so a further refresh fails.
	code, _ = s.do(http.MethodPost, "/api/v1/auth/refresh", nil, false)
	if code == http.StatusOK {
		t.Fatal("refresh after logout should fail")
	}
}

func TestLogoutInvalidatesSession(t *testing.T) {
	setup(t)
	_, email := seedUser(t, "Byebye", "employee")
	s := login(t, email, testPassword)

	if code, _ := s.do(http.MethodGet, "/api/v1/auth/me", nil, false); code != http.StatusOK {
		t.Fatalf("pre-logout /me should work, got %d", code)
	}
	if code, _ := s.do(http.MethodPost, "/api/v1/auth/logout", nil, true); code >= 300 {
		t.Fatalf("logout failed: %d", code)
	}
	if code, _ := s.do(http.MethodGet, "/api/v1/auth/me", nil, false); code != http.StatusUnauthorized {
		t.Fatalf("post-logout /me should be 401, got %d", code)
	}
}

func TestHashRoundTrip(t *testing.T) {
	h, err := password.Hash(testPassword)
	if err != nil {
		t.Fatal(err)
	}
	if !password.Compare(h, testPassword) {
		t.Fatal("hash should verify against original password")
	}
	if password.Compare(h, "other") {
		t.Fatal("hash must not verify wrong password")
	}
}
