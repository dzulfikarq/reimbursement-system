package auth

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"

	"github.com/mumtaz/reimbursement-system/backend/internal/config"
	"github.com/mumtaz/reimbursement-system/backend/internal/middleware"
	apperr "github.com/mumtaz/reimbursement-system/backend/internal/pkg/apperr"
	csrftoken "github.com/mumtaz/reimbursement-system/backend/internal/pkg/csrftoken"
	"github.com/mumtaz/reimbursement-system/backend/internal/pkg/response"
)

const (
	accessCookie  = "access_token"
	refreshCookie = "refresh_token"
	csrfCookie    = "csrf_token"
)

type Handler struct {
	cfg *config.Config
	svc *Service
}

func NewHandler(cfg *config.Config, svc *Service) *Handler {
	return &Handler{cfg: cfg, svc: svc}
}

// RegisterRoutes wires /auth endpoints. AuthN is applied per-route (login and
// refresh must stay public). guards applies rate limiting to credential
// endpoints (login/refresh) — pass nil to skip.
func (h *Handler) RegisterRoutes(v1 *gin.RouterGroup, guard ...gin.HandlerFunc) {
	g := v1.Group("/auth", guard...)
	{
		g.POST("/login", h.Login)
		g.POST("/refresh", h.Refresh)
		g.GET("/csrf", h.IssueCSRF)
		authed := g.Group("", middleware.AuthN(h.cfg.AppSecret))
		authed.POST("/logout", h.Logout)
		authed.GET("/me", h.Me)
	}
}

// Login godoc
// @Summary Log in
// @Description Sets access/refresh/csrf cookies on success.
// @Tags auth
// @Accept json
// @Produce json
// @Param body body LoginRequest true "Credentials"
// @Success 200 {object} response.Envelope
// @Failure 400 {object} response.Envelope
// @Failure 401 {object} response.Envelope
// @Failure 429 {object} response.Envelope
// @Router /auth/login [post]
func (h *Handler) Login(c *gin.Context) {
	var req LoginRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.Error(err)
		return
	}

	user, access, refresh, err := h.svc.Login(c.Request.Context(), req)
	if err != nil {
		c.Error(err)
		return
	}
	h.setSessionCookies(c, access, refresh)
	response.OK(c, gin.H{"user": user}, "Logged in")
}

// Refresh godoc
// @Summary Rotate session tokens
// @Description Single-use refresh; old refresh token is revoked.
// @Tags auth
// @Produce json
// @Success 200 {object} response.Envelope
// @Failure 401 {object} response.Envelope
// @Router /auth/refresh [post]
func (h *Handler) Refresh(c *gin.Context) {
	refresh, _ := c.Cookie(refreshCookie)
	if refresh == "" {
		response.Err(c, apperr.Unauthorized("Missing refresh token"))
		return
	}

	user, access, newRefresh, err := h.svc.Refresh(c.Request.Context(), refresh)
	if err != nil {
		h.clearSessionCookies(c)
		response.Err(c, err)
		return
	}
	h.setSessionCookies(c, access, newRefresh)
	response.OK(c, gin.H{"user": user}, "Session refreshed")
}

// Logout godoc
// @Summary Log out
// @Description Revokes the whole refresh chain and clears cookies.
// @Tags auth
// @Success 204 {object} response.Envelope
// @Router /auth/logout [post]
func (h *Handler) Logout(c *gin.Context) {
	refresh, _ := c.Cookie(refreshCookie)
	if err := h.svc.Logout(c.Request.Context(), refresh); err != nil {
		c.Error(err)
		return
	}
	h.clearSessionCookies(c)
	response.NoContent(c)
}

// Me godoc
// @Summary Current user
// @Tags auth
// @Produce json
// @Success 200 {object} response.Envelope
// @Failure 401 {object} response.Envelope
// @Router /auth/me [get]
func (h *Handler) Me(c *gin.Context) {
	userID, ok := c.Get(middleware.CtxUserID)
	if !ok {
		response.Err(c, apperr.Unauthorized("Authentication required"))
		return
	}
	uid, err := uuid.Parse(userID.(uuid.UUID).String())
	if err != nil {
		response.Err(c, apperr.Unauthorized("Invalid session"))
		return
	}
	user, err := h.svc.Me(c.Request.Context(), uid)
	if err != nil {
		c.Error(err)
		return
	}
	response.OK(c, gin.H{"user": user}, "OK")
}

// IssueCSRF godoc
// @Summary Issue CSRF cookie
// @Description Fresh CSRF token for the SPA after reload with expired csrf cookie.
// @Tags auth
// @Produce json
// @Success 200 {object} response.Envelope
// @Router /auth/csrf [get]
func (h *Handler) IssueCSRF(c *gin.Context) {
	token, err := csrftoken.Issue(h.cfg.AppSecret)
	if err != nil {
		c.Error(err)
		return
	}
	h.setCookie(c, csrfCookie, token, int(h.cfg.RefreshTTL.Seconds()), false)
	response.OK(c, gin.H{"token": token}, "OK")
}

// --- cookie helpers: attributes per docs/06 non-negotiable #1 ---

func (h *Handler) setSessionCookies(c *gin.Context, access, refresh string) {
	h.setCookie(c, accessCookie, access, int(h.cfg.AccessTTL.Seconds()), true)
	// Refresh scoped to the auth path only; strictest SameSite (see sameSiteFor).
	h.setCookieRaw(c, refreshCookie, refresh, int(h.cfg.RefreshTTL.Seconds()), true, "/api/v1/auth")

	token, err := csrftoken.Issue(h.cfg.AppSecret)
	if err == nil {
		h.setCookie(c, csrfCookie, token, int(h.cfg.RefreshTTL.Seconds()), false)
	}
}

func (h *Handler) clearSessionCookies(c *gin.Context) {
	h.setCookie(c, accessCookie, "", -1, true)
	h.setCookie(c, csrfCookie, "", -1, false)
	h.setCookieRaw(c, refreshCookie, "", -1, true, "/api/v1/auth")
}

func (h *Handler) setCookie(c *gin.Context, name, value string, maxAge int, httpOnly bool) {
	h.setCookieRaw(c, name, value, maxAge, httpOnly, "/")
}

func (h *Handler) setCookieRaw(c *gin.Context, name, value string, maxAge int, httpOnly bool, path string) {
	http.SetCookie(c.Writer, &http.Cookie{
		Name:     name,
		Value:    value,
		Path:     path,
		MaxAge:   maxAge,
		HttpOnly: httpOnly,
		Secure:   h.cfg.CookieSecure,
		SameSite: sameSiteFor(name),
	})
}

// Access rides every API path → SameSite=Lax. Refresh only ever travels to
// /api/v1/auth → strictest SameSite possible.
func sameSiteFor(name string) http.SameSite {
	if name == refreshCookie {
		return http.SameSiteStrictMode
	}
	return http.SameSiteLaxMode
}
