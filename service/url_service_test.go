package service

import (
	"errors"
	"testing"
	"time"

	"github.com/alicebob/miniredis/v2"
	"github.com/redis/go-redis/v9"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// mockURLRepo implements the URLService repo interface.
type mockURLRepo struct {
	store map[string]struct {
		longURL string
		clicks  int
		expiry  *time.Time
		userID  int
	}
	saveErr error
	getErr  error
}

func newMockURLRepo() *mockURLRepo {
	return &mockURLRepo{
		store: make(map[string]struct {
			longURL string
			clicks  int
			expiry  *time.Time
			userID  int
		}),
	}
}

func (m *mockURLRepo) Save(code, url string, expiry *time.Time, ip string, userID int) error {
	if m.saveErr != nil {
		return m.saveErr
	}
	m.store[code] = struct {
		longURL string
		clicks  int
		expiry  *time.Time
		userID  int
	}{longURL: url, expiry: expiry, userID: userID}
	return nil
}

func (m *mockURLRepo) Get(code string) (string, error) {
	if m.getErr != nil {
		return "", m.getErr
	}
	entry, ok := m.store[code]
	if !ok {
		return "", errors.New("not found")
	}
	return entry.longURL, nil
}

func (m *mockURLRepo) GetWithClicks(code string) (string, int, *time.Time, error) {
	entry, ok := m.store[code]
	if !ok {
		return "", 0, nil, errors.New("not found")
	}
	return entry.longURL, entry.clicks, entry.expiry, nil
}

func (m *mockURLRepo) IncrementClicks(code string) {
	if e, ok := m.store[code]; ok {
		e.clicks++
		m.store[code] = e
	}
}

func (m *mockURLRepo) UpdateClicks(code string, clicks int) {
	if e, ok := m.store[code]; ok {
		e.clicks += clicks
		m.store[code] = e
	}
}

func (m *mockURLRepo) GetUserURLs(userID int) ([]string, error) {
	var codes []string
	for code, e := range m.store {
		if e.userID == userID {
			codes = append(codes, code)
		}
	}
	return codes, nil
}

func (m *mockURLRepo) DeleteURL(code string, userID int) error {
	e, ok := m.store[code]
	if !ok || e.userID != userID {
		return errors.New("not found or forbidden")
	}
	delete(m.store, code)
	return nil
}

// newTestURLService spins up a miniredis instance and returns an URLService
// along with the underlying mini-redis server (call s.Close() after the test).
func newTestURLService(t *testing.T, repo *mockURLRepo) (*URLService, *miniredis.Miniredis) {
	t.Helper()
	mr, err := miniredis.Run()
	require.NoError(t, err)
	rdb := redis.NewClient(&redis.Options{Addr: mr.Addr()})
	svc := NewURLService(repo, rdb)
	return svc, mr
}

// ---------------------------------------------------------------------------
// CreateURL tests
// ---------------------------------------------------------------------------

func TestCreateURL_Success(t *testing.T) {
	repo := newMockURLRepo()
	svc, mr := newTestURLService(t, repo)
	defer mr.Close()

	code, err := svc.CreateURL("https://example.com", nil, nil, "127.0.0.1", 1)
	require.NoError(t, err)
	assert.NotEmpty(t, code)
}

func TestCreateURL_EmptyURL(t *testing.T) {
	repo := newMockURLRepo()
	svc, mr := newTestURLService(t, repo)
	defer mr.Close()

	_, err := svc.CreateURL("   ", nil, nil, "127.0.0.1", 1)
	require.Error(t, err)
	assert.Equal(t, "url is required", err.Error())
}

func TestCreateURL_ExpiredExpiry(t *testing.T) {
	repo := newMockURLRepo()
	svc, mr := newTestURLService(t, repo)
	defer mr.Close()

	past := time.Now().UTC().Add(-time.Hour)
	_, err := svc.CreateURL("https://example.com", nil, &past, "127.0.0.1", 1)
	require.Error(t, err)
	assert.Equal(t, "expiry must be in future", err.Error())
}

func TestCreateURL_WithAlias(t *testing.T) {
	repo := newMockURLRepo()
	svc, mr := newTestURLService(t, repo)
	defer mr.Close()

	alias := "my-alias"
	code, err := svc.CreateURL("https://example.com", &alias, nil, "127.0.0.1", 1)
	require.NoError(t, err)
	assert.Equal(t, alias, code)
}

func TestCreateURL_AliasDuplicate(t *testing.T) {
	repo := newMockURLRepo()
	svc, mr := newTestURLService(t, repo)
	defer mr.Close()

	alias := "dup"
	_, err := svc.CreateURL("https://example.com", &alias, nil, "127.0.0.1", 1)
	require.NoError(t, err)

	_, err = svc.CreateURL("https://other.com", &alias, nil, "127.0.0.1", 2)
	require.Error(t, err)
	assert.Equal(t, "alias already exists", err.Error())
}

func TestCreateURL_WithFutureExpiry(t *testing.T) {
	repo := newMockURLRepo()
	svc, mr := newTestURLService(t, repo)
	defer mr.Close()

	future := time.Now().UTC().Add(time.Hour)
	code, err := svc.CreateURL("https://example.com", nil, &future, "127.0.0.1", 1)
	require.NoError(t, err)
	assert.NotEmpty(t, code)
}

func TestCreateURL_AliasWithFutureExpiry(t *testing.T) {
	repo := newMockURLRepo()
	svc, mr := newTestURLService(t, repo)
	defer mr.Close()

	alias := "timed-alias"
	future := time.Now().UTC().Add(time.Hour)
	code, err := svc.CreateURL("https://example.com", &alias, &future, "127.0.0.1", 1)
	require.NoError(t, err)
	assert.Equal(t, alias, code)
}

func TestCreateURL_RepoSaveError(t *testing.T) {
	repo := newMockURLRepo()
	repo.saveErr = errors.New("db write error")
	svc, mr := newTestURLService(t, repo)
	defer mr.Close()

	alias := "err-alias"
	_, err := svc.CreateURL("https://example.com", &alias, nil, "127.0.0.1", 1)
	require.Error(t, err)
	assert.Equal(t, "db write error", err.Error())
}

// ---------------------------------------------------------------------------
// GetURL tests
// ---------------------------------------------------------------------------

func TestGetURL_CacheHit(t *testing.T) {
	repo := newMockURLRepo()
	svc, mr := newTestURLService(t, repo)
	defer mr.Close()

	// prime the cache directly
	mr.Set("url:abc", "https://cached.com")

	url, err := svc.GetURL("abc")
	require.NoError(t, err)
	assert.Equal(t, "https://cached.com", url)
}

func TestGetURL_CacheMiss_DBHit(t *testing.T) {
	repo := newMockURLRepo()
	svc, mr := newTestURLService(t, repo)
	defer mr.Close()

	require.NoError(t, repo.Save("xyz", "https://db.com", nil, "127.0.0.1", 1))

	url, err := svc.GetURL("xyz")
	require.NoError(t, err)
	assert.Equal(t, "https://db.com", url)
}

func TestGetURL_NotFound(t *testing.T) {
	repo := newMockURLRepo()
	svc, mr := newTestURLService(t, repo)
	defer mr.Close()

	_, err := svc.GetURL("missing")
	require.Error(t, err)
}

func TestGetURL_Expired(t *testing.T) {
	repo := newMockURLRepo()
	svc, mr := newTestURLService(t, repo)
	defer mr.Close()

	past := time.Now().UTC().Add(-time.Hour)
	require.NoError(t, repo.Save("old", "https://old.com", &past, "127.0.0.1", 1))

	_, err := svc.GetURL("old")
	require.Error(t, err)
	assert.Equal(t, "link expired", err.Error())
}

func TestGetURL_CacheMiss_FutureExpiry(t *testing.T) {
	repo := newMockURLRepo()
	svc, mr := newTestURLService(t, repo)
	defer mr.Close()

	future := time.Now().UTC().Add(time.Hour)
	require.NoError(t, repo.Save("live", "https://live.com", &future, "127.0.0.1", 1))

	url, err := svc.GetURL("live")
	require.NoError(t, err)
	assert.Equal(t, "https://live.com", url)
}

func TestGetURL_CachePopulatedOnMiss(t *testing.T) {
	repo := newMockURLRepo()
	svc, mr := newTestURLService(t, repo)
	defer mr.Close()

	require.NoError(t, repo.Save("pop", "https://populate.com", nil, "127.0.0.1", 1))

	_, err := svc.GetURL("pop")
	require.NoError(t, err)

	// the value should now be cached in miniredis
	val, err := mr.Get("url:pop")
	require.NoError(t, err)
	assert.Equal(t, "https://populate.com", val)
}

// ---------------------------------------------------------------------------
// CheckAlias tests
// ---------------------------------------------------------------------------

func TestCheckAlias_Available(t *testing.T) {
	repo := newMockURLRepo()
	svc, mr := newTestURLService(t, repo)
	defer mr.Close()

	exists, err := svc.CheckAlias("free-code")
	require.NoError(t, err)
	assert.False(t, exists)
}

func TestCheckAlias_Taken(t *testing.T) {
	repo := newMockURLRepo()
	svc, mr := newTestURLService(t, repo)
	defer mr.Close()

	require.NoError(t, repo.Save("taken", "https://taken.com", nil, "127.0.0.1", 1))

	exists, err := svc.CheckAlias("taken")
	require.NoError(t, err)
	assert.True(t, exists)
}

// ---------------------------------------------------------------------------
// DeleteURL tests
// ---------------------------------------------------------------------------

func TestDeleteURL_Success(t *testing.T) {
	repo := newMockURLRepo()
	svc, mr := newTestURLService(t, repo)
	defer mr.Close()

	require.NoError(t, repo.Save("del", "https://del.com", nil, "127.0.0.1", 42))

	err := svc.DeleteURL("del", 42)
	require.NoError(t, err)
}

func TestDeleteURL_WrongUser(t *testing.T) {
	repo := newMockURLRepo()
	svc, mr := newTestURLService(t, repo)
	defer mr.Close()

	require.NoError(t, repo.Save("mine", "https://mine.com", nil, "127.0.0.1", 1))

	err := svc.DeleteURL("mine", 99)
	require.Error(t, err)
}

// ---------------------------------------------------------------------------
// GetUserURLs tests
// ---------------------------------------------------------------------------

func TestGetUserURLs_Empty(t *testing.T) {
	repo := newMockURLRepo()
	svc, mr := newTestURLService(t, repo)
	defer mr.Close()

	urls, err := svc.GetUserURLs(99)
	require.NoError(t, err)
	assert.Empty(t, urls)
}

func TestGetUserURLs_ReturnsOwned(t *testing.T) {
	repo := newMockURLRepo()
	svc, mr := newTestURLService(t, repo)
	defer mr.Close()

	require.NoError(t, repo.Save("u1c1", "https://a.com", nil, "127.0.0.1", 1))
	require.NoError(t, repo.Save("u1c2", "https://b.com", nil, "127.0.0.1", 1))
	require.NoError(t, repo.Save("u2c1", "https://c.com", nil, "127.0.0.1", 2))

	urls, err := svc.GetUserURLs(1)
	require.NoError(t, err)
	assert.Len(t, urls, 2)
	assert.ElementsMatch(t, []string{"u1c1", "u1c2"}, urls)
}
