package worker

import (
	"context"
	"log"
	"time"
)

type ExpiryRepo interface {
	DeleteExpired() error
}

func StartExpiryCleanup(ctx context.Context, repo ExpiryRepo) {
	ticker := time.NewTicker(10 * time.Second)

	go func() {
		defer ticker.Stop()

		for {
			select {
			case <-ticker.C:
				log.Println("running expiry cleanup...")

				if err := repo.DeleteExpired(); err != nil {
					log.Println("cleanup error:", err)
				} else {
					log.Println("expired URLs cleaned")
				}

			case <-ctx.Done():
				log.Println("expiry cleanup stopped")
				return
			}
		}
	}()
}
