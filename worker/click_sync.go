package worker

import (
	"context"
	"strings"
	"time"

	"github.com/redis/go-redis/v9"
)

type Repo interface {
	UpdateClicks(code string, clicks int)
}

func StartClickSync(rdb *redis.Client, repo Repo) {

	ctx := context.Background()

	go func() {
		for {
			keys, _ := rdb.Keys(ctx, "url:*:clicks").Result()

			for _, key := range keys {

				val, _ := rdb.Get(ctx, key).Int()

				if val > 0 {
					parts := strings.Split(key, ":")
					code := parts[1]

					repo.UpdateClicks(code, val)

					rdb.Del(ctx, key)
				}
			}

			time.Sleep(10 * time.Second)
		}
	}()
}
