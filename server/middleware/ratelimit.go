package middleware

import (
	"net/http"
	"sync"
	"time"

	"github.com/gin-gonic/gin"
)

type RateLimitTier struct {
	Rate   int
	Window time.Duration
}

var (
	TierUpload   = RateLimitTier{Rate: 10, Window: time.Minute}
	TierAdmin    = RateLimitTier{Rate: 60, Window: time.Minute}
	TierLogin    = RateLimitTier{Rate: 5, Window: time.Minute}
	TierDownload = RateLimitTier{Rate: 20, Window: time.Minute}
)

type entry struct {
	timestamps []time.Time
}

type RateLimiter struct {
	mu        sync.RWMutex
	ips       map[string]*entry
	tier      RateLimitTier
	skipLocal bool
	stopCh    chan struct{}
}

func NewRateLimiter(tier RateLimitTier, skipLocal bool) *RateLimiter {
	rl := &RateLimiter{
		ips:       make(map[string]*entry),
		tier:      tier,
		skipLocal: skipLocal,
		stopCh:    make(chan struct{}),
	}
	go rl.cleanupLoop()
	return rl
}

func (rl *RateLimiter) Middleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		ip := c.ClientIP()
		if rl.skipLocal && (ip == "127.0.0.1" || ip == "::1") {
			c.Next()
			return
		}

		now := time.Now()
		windowStart := now.Add(-rl.tier.Window)

		rl.mu.Lock()

		e, exists := rl.ips[ip]
		if !exists {
			e = &entry{}
			rl.ips[ip] = e
		}

		// Prune old timestamps
		valid := e.timestamps[:0]
		for _, t := range e.timestamps {
			if t.After(windowStart) {
				valid = append(valid, t)
			}
		}
		e.timestamps = valid

		if len(e.timestamps) >= rl.tier.Rate {
			rl.mu.Unlock()
			c.AbortWithStatusJSON(http.StatusTooManyRequests, gin.H{
				"code":    -1,
				"message": "请求过于频繁，请稍后再试",
			})
			return
		}

		e.timestamps = append(e.timestamps, now)
		rl.mu.Unlock()

		c.Next()
	}
}

func (rl *RateLimiter) cleanupLoop() {
	ticker := time.NewTicker(5 * time.Minute)
	defer ticker.Stop()
	for {
		select {
		case <-ticker.C:
			rl.prune()
		case <-rl.stopCh:
			return
		}
	}
}

func (rl *RateLimiter) prune() {
	rl.mu.Lock()
	defer rl.mu.Unlock()

	windowStart := time.Now().Add(-rl.tier.Window)
	for ip, e := range rl.ips {
		valid := e.timestamps[:0]
		for _, t := range e.timestamps {
			if t.After(windowStart) {
				valid = append(valid, t)
			}
		}
		if len(valid) == 0 {
			delete(rl.ips, ip)
		} else {
			e.timestamps = valid
		}
	}
}
