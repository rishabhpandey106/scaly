package service

import (
	"errors"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// mockUserRepo implements UserRepository for testing.
type mockUserRepo struct {
	users map[string]struct {
		id       int
		password string
	}
	createErr    error
	getByEmailFn func(email string) (int, string, error)
}

func newMockUserRepo() *mockUserRepo {
	return &mockUserRepo{
		users: make(map[string]struct {
			id       int
			password string
		}),
	}
}

func (m *mockUserRepo) Create(email, hashedPassword string) (int, error) {
	if m.createErr != nil {
		return 0, m.createErr
	}
	id := len(m.users) + 1
	m.users[email] = struct {
		id       int
		password string
	}{id: id, password: hashedPassword}
	return id, nil
}

func (m *mockUserRepo) GetByEmail(email string) (int, string, error) {
	if m.getByEmailFn != nil {
		return m.getByEmailFn(email)
	}
	u, ok := m.users[email]
	if !ok {
		return 0, "", errors.New("not found")
	}
	return u.id, u.password, nil
}

// helpers
func newTestAuthService(repo UserRepository) *AuthService {
	return NewAuthService(repo, "test-secret-key", time.Hour)
}

// ---------------------------------------------------------------------------
// Signup tests
// ---------------------------------------------------------------------------

func TestSignup_Success(t *testing.T) {
	repo := newMockUserRepo()
	svc := newTestAuthService(repo)

	userID, token, err := svc.Signup("alice@example.com", "password123")

	require.NoError(t, err)
	assert.Positive(t, userID)
	assert.NotEmpty(t, token)
}

func TestSignup_DuplicateEmail(t *testing.T) {
	repo := newMockUserRepo()
	svc := newTestAuthService(repo)

	_, _, err := svc.Signup("alice@example.com", "password123")
	require.NoError(t, err)

	_, _, err = svc.Signup("alice@example.com", "anotherpass")
	require.Error(t, err)
	assert.Equal(t, "email already registered", err.Error())
}

func TestSignup_RepoCreateError(t *testing.T) {
	repo := newMockUserRepo()
	repo.createErr = errors.New("db error")
	svc := newTestAuthService(repo)

	// make GetByEmail return not-found so signup proceeds past duplicate check
	repo.getByEmailFn = func(email string) (int, string, error) {
		return 0, "", errors.New("not found")
	}

	_, _, err := svc.Signup("bob@example.com", "password123")
	require.Error(t, err)
	assert.Equal(t, "db error", err.Error())
}

func TestSignup_TokenIsValid(t *testing.T) {
	repo := newMockUserRepo()
	svc := newTestAuthService(repo)

	userID, token, err := svc.Signup("carol@example.com", "password123")
	require.NoError(t, err)

	parsedID, err := svc.ValidateToken(token)
	require.NoError(t, err)
	assert.Equal(t, userID, parsedID)
}

// ---------------------------------------------------------------------------
// Login tests
// ---------------------------------------------------------------------------

func TestLogin_Success(t *testing.T) {
	repo := newMockUserRepo()
	svc := newTestAuthService(repo)

	signupID, _, err := svc.Signup("dave@example.com", "mypassword")
	require.NoError(t, err)

	loginID, token, err := svc.Login("dave@example.com", "mypassword")
	require.NoError(t, err)
	assert.Equal(t, signupID, loginID)
	assert.NotEmpty(t, token)
}

func TestLogin_WrongPassword(t *testing.T) {
	repo := newMockUserRepo()
	svc := newTestAuthService(repo)

	_, _, err := svc.Signup("eve@example.com", "correctpass")
	require.NoError(t, err)

	_, _, err = svc.Login("eve@example.com", "wrongpass")
	require.Error(t, err)
	assert.Equal(t, "invalid credentials", err.Error())
}

func TestLogin_UnknownEmail(t *testing.T) {
	repo := newMockUserRepo()
	svc := newTestAuthService(repo)

	_, _, err := svc.Login("nobody@example.com", "pass")
	require.Error(t, err)
	assert.Equal(t, "invalid credentials", err.Error())
}

func TestLogin_TokenIsValid(t *testing.T) {
	repo := newMockUserRepo()
	svc := newTestAuthService(repo)

	id, _, _ := svc.Signup("frank@example.com", "pass1234")
	_, token, err := svc.Login("frank@example.com", "pass1234")
	require.NoError(t, err)

	parsedID, err := svc.ValidateToken(token)
	require.NoError(t, err)
	assert.Equal(t, id, parsedID)
}

// ---------------------------------------------------------------------------
// ValidateToken tests
// ---------------------------------------------------------------------------

func TestValidateToken_InvalidToken(t *testing.T) {
	svc := newTestAuthService(newMockUserRepo())
	_, err := svc.ValidateToken("not.a.valid.token")
	assert.Error(t, err)
}

func TestValidateToken_WrongSecret(t *testing.T) {
	repo := newMockUserRepo()
	svc1 := NewAuthService(repo, "secret-one", time.Hour)
	svc2 := NewAuthService(repo, "secret-two", time.Hour)

	_, token, err := svc1.Signup("grace@example.com", "pass1234")
	require.NoError(t, err)

	_, err = svc2.ValidateToken(token)
	assert.Error(t, err)
}

func TestValidateToken_ExpiredToken(t *testing.T) {
	repo := newMockUserRepo()
	// token expiry of -1 second → already expired when issued
	svc := NewAuthService(repo, "secret", -time.Second)

	_, _, err := svc.Signup("heidi@example.com", "pass1234")
	require.NoError(t, err)

	// issue an expired token manually via generateToken
	token, err := svc.generateToken(1)
	require.NoError(t, err)

	_, err = svc.ValidateToken(token)
	assert.Error(t, err)
}

func TestValidateToken_EmptyString(t *testing.T) {
	svc := newTestAuthService(newMockUserRepo())
	_, err := svc.ValidateToken("")
	assert.Error(t, err)
}
