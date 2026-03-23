package app

import (
	"fmt"
	"net/http"
	"time"

	"github.com/DrummDaddy/task_service/internal/auth"
	"github.com/DrummDaddy/task_service/internal/httpx"
	"github.com/redis/go-redis/v9"
)

func NewRateLimitMiddleware(rdb *redis.Client, perUserPerMinute int) func(next http.Handler) http.Handler {
	if perUserPerMinute <= 0 {
		perUserPerMinute = 100
	}
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			userID, ok := auth.UserIDFromContext(r.Context())
			if !ok {
				httpx.Error(w, http.StatusUnauthorized, "unauthorized")
				return
			}

			now := time.Now().Unix()
			bucket := now / 60
			key := fmt.Sprintf("%d-%d", userID, bucket)

			pipe := rdb.Pipeline()
			incr := pipe.Incr(r.Context(), key)
			pipe.Expire(r.Context(), key, 70*time.Second)
			_, err := pipe.Exec(r.Context())
			if err != nil {
				httpx.Error(w, http.StatusServiceUnavailable, "redis pipeline failed")
				return
			}
			if int(incr.Val()) > perUserPerMinute {
				httpx.Error(w, http.StatusTooManyRequests, "too many requests")
				return
			}
			next.ServeHTTP(w, r)
		})
	}
}
