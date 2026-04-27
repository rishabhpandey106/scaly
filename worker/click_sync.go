package worker

import (
	"context"
	"log"
	"strings"
	"time"

	"github.com/redis/go-redis/v9"
)

type Repo interface {
	UpdateClicks(code string, clicks int)
}

func StartClickSync(ctx context.Context, rdb *redis.Client, repo Repo) {
	ticker := time.NewTicker(10 * time.Second)

	go func() {
		defer ticker.Stop()

		for {
			select {
			case <-ticker.C:

				iter := rdb.Scan(ctx, 0, "url:*:clicks", 100).Iterator()

				for iter.Next(ctx) {
					key := iter.Val()

					val, err := rdb.Get(ctx, key).Int()
					if err != nil {
						log.Println("error getting value:", err)
						continue
					}

					if val > 0 {
						parts := strings.Split(key, ":")
						if len(parts) < 3 {
							continue
						}

						code := parts[1]

						repo.UpdateClicks(code, val)

						rdb.Del(ctx, key)
					}
				}

				if err := iter.Err(); err != nil {
					log.Println("scan error:", err)
				}

			case <-ctx.Done():
				log.Println("click sync stopped")
				return
			}
		}
	}()
}
