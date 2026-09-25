package health

import (
	"context"
	"database/sql"
	"encoding/json"
	"net/http"
	"os"
	"time"

	"github.com/redis/go-redis/v9"
)

type Status string

const (
	StatusUp       Status = "up"
	StatusDown     Status = "down"
	StatusDegraded Status = "degraded"
)

type ComponentHealth struct {
	Status  Status `json:"status"`
	Message string `json:"message,omitempty"`
	Latency string `json:"latency,omitempty"`
}

type HealthResponse struct {
	Status     Status                     `json:"status"`
	Version    string                     `json:"version"`
	Timestamp  time.Time                  `json:"timestamp"`
	Components map[string]ComponentHealth `json:"components"`
}

type Checker struct {
	db  *sql.DB
	rdb *redis.Client
}

func NewChecker(db *sql.DB, rdb *redis.Client) *Checker {
	return &Checker{db: db, rdb: rdb}
}

func (c *Checker) Handle(w http.ResponseWriter, r *http.Request) {
	ctx, cancel := context.WithTimeout(r.Context(), 5*time.Second)
	defer cancel()

	response := HealthResponse{
		Status:     StatusUp,
		Version:    getEnv("APP_VERSION", "dev"),
		Timestamp:  time.Now().UTC(),
		Components: make(map[string]ComponentHealth),
	}

	dbHealth := c.checkDatabase(ctx)
	response.Components["database"] = dbHealth
	if dbHealth.Status == StatusDown {
		response.Status = StatusDown
	}

	redisHealth := c.checkRedis(ctx)
	response.Components["redis"] = redisHealth
	if redisHealth.Status == StatusDown {
		response.Status = StatusDown
	} else if redisHealth.Status == StatusDegraded && response.Status == StatusUp {
		response.Status = StatusDegraded
	}

	w.Header().Set("Content-Type", "application/json")
	if response.Status == StatusDown {
		w.WriteHeader(http.StatusServiceUnavailable)
	} else {
		w.WriteHeader(http.StatusOK)
	}

	_ = json.NewEncoder(w).Encode(response)
}

func (c *Checker) HandleLive(w http.ResponseWriter, _ *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	_ = json.NewEncoder(w).Encode(map[string]string{"status": "alive"})
}

func (c *Checker) checkDatabase(ctx context.Context) ComponentHealth {
	if c.db == nil {
		return ComponentHealth{Status: StatusDown, Message: "database not configured"}
	}

	start := time.Now()
	err := c.db.PingContext(ctx)
	latency := time.Since(start)

	if err != nil {
		return ComponentHealth{
			Status:  StatusDown,
			Message: "database ping failed: " + err.Error(),
			Latency: latency.String(),
		}
	}

	stats := c.db.Stats()
	if stats.OpenConnections > 20 {
		return ComponentHealth{
			Status:  StatusDegraded,
			Message: "high connection count",
			Latency: latency.String(),
		}
	}

	return ComponentHealth{
		Status:  StatusUp,
		Latency: latency.String(),
	}
}

func (c *Checker) checkRedis(ctx context.Context) ComponentHealth {
	if c.rdb == nil {
		return ComponentHealth{Status: StatusDegraded, Message: "redis not configured (using in-memory fallback)"}
	}

	start := time.Now()
	err := c.rdb.Ping(ctx).Err()
	latency := time.Since(start)

	if err != nil {
		return ComponentHealth{
			Status:  StatusDown,
			Message: "redis ping failed: " + err.Error(),
			Latency: latency.String(),
		}
	}

	return ComponentHealth{
		Status:  StatusUp,
		Latency: latency.String(),
	}
}

func getEnv(key, fallback string) string {
	v := os.Getenv(key)
	if v != "" {
		return v
	}
	return fallback
}
