package middleware

import (
	"net/http"
	"sync"
	"time"

	"github.com/gin-gonic/gin"

	"github.com/alvor-technologies/iag-platform-go/apierr"
)

// rateLimiter is a best-effort, per-pod, per-client-IP fixed-window limiter.
// Production deployments behind multiple replicas should be aware that the
// effective ceiling is `limit * replicas`. For strict global limits, swap
// this for a Redis sliding-window implementation.
type rateLimiter struct {
	mu      sync.Mutex
	entries map[string]*ipLimiter
	limit   int
}

type ipLimiter struct {
	mu     sync.Mutex
	window time.Time
	count  int
	limit  int
}

func newRateLimiter(perMinute int) *rateLimiter {
	if perMinute < 1 {
		perMinute = 120
	}
	rl := &rateLimiter{
		entries: make(map[string]*ipLimiter),
		limit:   perMinute,
	}
	return rl
}

// startEvictor periodically removes IP entries that haven't been touched in
// the last 5 minutes. Without this the entries map leaked one entry per
// unique client IP for the lifetime of the process.
func (rl *rateLimiter) startEvictor(interval, idle time.Duration, stop <-chan struct{}) {
	go func() {
		t := time.NewTicker(interval)
		defer t.Stop()
		for {
			select {
			case <-stop:
				return
			case now := <-t.C:
				rl.mu.Lock()
				for k, e := range rl.entries {
					e.mu.Lock()
					stale := now.Sub(e.window) > idle
					e.mu.Unlock()
					if stale {
						delete(rl.entries, k)
					}
				}
				rl.mu.Unlock()
			}
		}
	}()
}

func (rl *rateLimiter) allow(key string) bool {
	now := time.Now()
	rl.mu.Lock()
	e, ok := rl.entries[key]
	if !ok {
		e = &ipLimiter{window: now, limit: rl.limit}
		rl.entries[key] = e
	}
	rl.mu.Unlock()

	e.mu.Lock()
	defer e.mu.Unlock()
	if now.Sub(e.window) >= time.Minute {
		e.window = now
		e.count = 0
	}
	if e.count >= e.limit {
		return false
	}
	e.count++
	return true
}

// GinRateLimit installs a fixed-window IP rate limit. Entries are evicted
// after 5 minutes of inactivity so the per-IP map cannot grow unbounded.
// The evictor goroutine is started once per middleware instance.
func GinRateLimit(perMinute int) gin.HandlerFunc {
	rl := newRateLimiter(perMinute)
	rl.startEvictor(time.Minute, 5*time.Minute, nil)
	return func(c *gin.Context) {
		if !rl.allow(c.ClientIP()) {
			c.AbortWithStatusJSON(http.StatusTooManyRequests, gin.H{
				"error": gin.H{"code": apierr.CodeTooManyRequests, "message": "rate limit exceeded"},
			})
			return
		}
		c.Next()
	}
}
