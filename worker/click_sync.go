package worker

import (
	"context"
	"strconv"
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

				val, _ := rdb.Get(ctx, key).Result()
				clicks, _ := strconv.Atoi(val)

				parts := strings.Split(key, ":")
				code := parts[1]

				repo.UpdateClicks(code, clicks)

				// reset counter after sync
				rdb.Del(ctx, key)
			}

			time.Sleep(10 * time.Second)
		}
	}()
}
