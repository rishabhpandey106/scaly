package worker

import (
	"context"
	"sync"
	"testing"
	"time"

	"github.com/alicebob/miniredis/v2"
	"github.com/redis/go-redis/v9"
	"github.com/stretchr/testify/assert"
)

// mockClickRepo records UpdateClicks calls.
type mockClickRepo struct {
	mu     sync.Mutex
	calls  []struct{ code string; clicks int }
	deleted []string
}

func (m *mockClickRepo) UpdateClicks(code string, clicks int) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.calls = append(m.calls, struct{ code string; clicks int }{code, clicks})
}

func TestStartClickSync_StopsOnContextCancel(t *testing.T) {
	mr, err := miniredis.Run()
	assert.NoError(t, err)
	defer mr.Close()

	rdb := redis.NewClient(&redis.Options{Addr: mr.Addr()})
	defer rdb.Close()

	repo := &mockClickRepo{}
	ctx, cancel := context.WithCancel(context.Background())

	StartClickSync(ctx, rdb, repo)

	cancel()

	// Give the goroutine time to process the cancellation.
	time.Sleep(20 * time.Millisecond)

	// After cancel, no panics and the goroutine should have exited.
	repo.mu.Lock()
	countAfterCancel := len(repo.calls)
	repo.mu.Unlock()

	time.Sleep(20 * time.Millisecond)

	repo.mu.Lock()
	defer repo.mu.Unlock()
	assert.Equal(t, countAfterCancel, len(repo.calls), "no more UpdateClicks calls after cancel")
}

func TestStartClickSync_DoesNotPanicWithEmptyRedis(t *testing.T) {
	mr, err := miniredis.Run()
	assert.NoError(t, err)
	defer mr.Close()

	rdb := redis.NewClient(&redis.Options{Addr: mr.Addr()})
	defer rdb.Close()

	repo := &mockClickRepo{}
	ctx, cancel := context.WithCancel(context.Background())

	assert.NotPanics(t, func() {
		StartClickSync(ctx, rdb, repo)
		time.Sleep(20 * time.Millisecond)
		cancel()
		time.Sleep(20 * time.Millisecond)
	})
}

func TestStartClickSync_HandlesInvalidKeyFormat(t *testing.T) {
	mr, err := miniredis.Run()
	assert.NoError(t, err)
	defer mr.Close()

	rdb := redis.NewClient(&redis.Options{Addr: mr.Addr()})
	defer rdb.Close()

	// Seed a malformed click key (only 2 parts instead of 3).
	mr.Set("url:clicks", "5")

	repo := &mockClickRepo{}
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// Should not panic even with a malformed key.
	assert.NotPanics(t, func() {
		StartClickSync(ctx, rdb, repo)
		time.Sleep(20 * time.Millisecond)
	})
}
