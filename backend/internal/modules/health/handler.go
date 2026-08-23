package health

import (
	"context"
	"log/slog"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/minio/minio-go/v7"
	goredis "github.com/redis/go-redis/v9"
	"gorm.io/gorm"

	"github.com/mumtaz/reimbursement-system/backend/internal/pkg/response"
)

// Deps carries everything the health endpoint probes.
type Deps struct {
	DB          *gorm.DB
	Redis       *goredis.Client
	Minio       *minio.Client
	MinioBucket string
}

// Handler serves GET /healthz: liveness always, readiness flags for
// postgres/redis/minio. 200 when all green, 503 otherwise.
func Handler(d Deps) gin.HandlerFunc {
	type svc struct{ Status string `json:"status"` }
	return func(c *gin.Context) {
		ctx, cancel := context.WithTimeout(c.Request.Context(), 3*time.Second)
		defer cancel()

		services := gin.H{}
		check := func(name string, ok bool) {
			status := "up"
			if !ok {
				status = "down"
			}
			services[name] = svc{Status: status}
		}

		pgOK := d.DB != nil && d.DB.WithContext(ctx).Exec("SELECT 1").Error == nil
		rdOK := d.Redis != nil && d.Redis.Ping(ctx).Err() == nil
		mnOK := false
		if d.Minio != nil {
			exists, err := d.Minio.BucketExists(ctx, d.MinioBucket)
			mnOK = err == nil && exists
		}

		check("postgres", pgOK)
		check("redis", rdOK)
		check("minio", mnOK)

		if !(pgOK && rdOK && mnOK) {
			slog.WarnContext(ctx, "health_check_degraded",
				"postgres", pgOK, "redis", rdOK, "minio", mnOK)
			c.JSON(http.StatusServiceUnavailable, gin.H{
				"success": true,
				"data":    gin.H{"status": "degraded", "services": services},
			})
			return
		}
		response.OK(c, gin.H{"status": "ok", "services": services}, "Healthy")
	}
}
