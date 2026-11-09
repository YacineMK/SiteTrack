package auth

import (
	"context"
	"errors"
	"sync"
	"testing"
	"time"

	"github.com/google/uuid"

	"github.com/YacineMK/SiteTrack/kit/hash"
	kitjwt "github.com/YacineMK/SiteTrack/kit/jwt"
)

func newTestService() (*Service, *memoryUserRepository, *memoryTokenBlacklist, *kitjwt.Manager) {
	repo := newMemoryUserRepository()
	blacklist := newMemoryTokenBlacklist()
	manager := kitjwt.New("test-secret", 15*time.Minute, 24*time.Hour)
	return NewService(repo, manager, blacklist), repo, blacklist, manager
}

func registerAlice(t *testing.T, svc *Service) UserResponse {
	t.Helper()
	user, err := svc.Register(context.Background(), RegisterRequest{
		Name:     "Alice",
		Email:    "alice@example.com",
		Password: "password123",
		AppRole:  "user",
	})
	if err != nil {
		t.Fatalf("Register() error = %v", err)
	}
	return user
}

func TestRegisterStoresHashedPassword(t *testing.T) {
	svc, repo, _, _ := newTestService()

	user := registerAlice(t, svc)

	if user.Name != "Alice" {
		t.Fatalf("Name = %q, want %q", user.Name, "Alice")
	}
	if user.Email != "alice@example.com" {
		t.Fatalf("Email = %q, want %q", user.Email, "alice@example.com")
	}
	if user.AppRole != "user" {
		t.Fatalf("AppRole = %q, want %q", user.AppRole, "user")
	}
	if user.ID == "" {
		t.Fatal("ID is empty")
	}

	stored, err := repo.GetByEmail(context.Background(), "alice@example.com")
	if err != nil {
		t.Fatalf("GetByEmail() error = %v", err)
	}
	if stored.PasswordHash == "password123" {
		t.Fatal("password was stored in plain text")
	}
	match, err := hash.Verify("password123", stored.PasswordHash)
	if err != nil {
		t.Fatalf("Verify() error = %v", err)
	}
	if !match {
		t.Fatal("stored hash does not match the original password")
	}
	if stored.TokenVersion != 1 {
		t.Fatalf("TokenVersion = %d, want 1", stored.TokenVersion)
	}
}

func TestRegisterNormalizesEmail(t *testing.T) {
	svc, _, _, _ := newTestService()

	user, err := svc.Register(context.Background(), RegisterRequest{
		Name:     "  Alice  ",
		Email:    "  Alice@Example.COM ",
		Password: "password123",
		AppRole:  "user",
	})
	if err != nil {
		t.Fatalf("Register() error = %v", err)
	}
	if user.Name != "Alice" {
		t.Fatalf("Name = %q, want trimmed %q", user.Name, "Alice")
	}
	if user.Email != "alice@example.com" {
		t.Fatalf("Email = %q, want normalized %q", user.Email, "alice@example.com")
	}
}

func TestRegisterRejectsDuplicateEmail(t *testing.T) {
	svc, _, _, _ := newTestService()
	registerAlice(t, svc)

	_, err := svc.Register(context.Background(), RegisterRequest{
		Name:     "Other Alice",
		Email:    "alice@example.com",
		Password: "password456",
		AppRole:  "user",
	})
	if !errors.Is(err, ErrEmailExists) {
		t.Fatalf("Register() error = %v, want ErrEmailExists", err)
	}
}

func TestRegisterRejectsInvalidInput(t *testing.T) {
	tests := []struct {
		name string
		req  RegisterRequest
	}{
		{name: "empty name", req: RegisterRequest{Name: "", Email: "a@example.com", Password: "password123", AppRole: "user"}},
		{name: "invalid email", req: RegisterRequest{Name: "A", Email: "not-an-email", Password: "password123", AppRole: "user"}},
		{name: "short password", req: RegisterRequest{Name: "A", Email: "a@example.com", Password: "short", AppRole: "user"}},
		{name: "empty role", req: RegisterRequest{Name: "A", Email: "a@example.com", Password: "password123", AppRole: ""}},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			svc, repo, _, _ := newTestService()
			_, err := svc.Register(context.Background(), tt.req)
			if !errors.Is(err, ErrInvalidInput) {
				t.Fatalf("Register() error = %v, want ErrInvalidInput", err)
			}
			if repo.count() != 0 {
				t.Fatal("invalid registration must not persist a user")
			}
		})
	}
}

func TestLoginReturnsValidTokenPair(t *testing.T) {
	svc, _, _, manager := newTestService()
	user := registerAlice(t, svc)

	resp, err := svc.Login(context.Background(), LoginRequest{
		Email:    "alice@example.com",
		Password: "password123",
	})
	if err != nil {
		t.Fatalf("Login() error = %v", err)
	}
	if resp.AccessToken == "" || resp.RefreshToken == "" {
		t.Fatal("Login() returned an empty token")
	}
	if !resp.AccessExpiresAt.After(time.Now()) || !resp.RefreshExpiresAt.After(resp.AccessExpiresAt) {
		t.Fatal("Login() returned invalid token expirations")
	}
	if resp.User.ID != user.ID {
		t.Fatalf("User.ID = %q, want %q", resp.User.ID, user.ID)
	}

	claims, err := manager.ParseAccess(resp.AccessToken)
	if err != nil {
		t.Fatalf("ParseAccess() error = %v", err)
	}
	if claims.UserID != user.ID || claims.AppRole != "user" || claims.TokenVersion != 1 {
		t.Fatalf("unexpected access claims: %+v", claims)
	}

	if _, err := manager.ParseRefresh(resp.RefreshToken); err != nil {
		t.Fatalf("ParseRefresh() error = %v", err)
	}
}

func TestLoginUsesGenericErrorForBadCredentials(t *testing.T) {
	tests := []struct {
		name string
		req  LoginRequest
	}{
		{name: "wrong password", req: LoginRequest{Email: "alice@example.com", Password: "wrong-password"}},
		{name: "unknown email", req: LoginRequest{Email: "nobody@example.com", Password: "password123"}},
	}

	var messages []string
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			svc, _, _, _ := newTestService()
			registerAlice(t, svc)

			_, err := svc.Login(context.Background(), tt.req)
			if !errors.Is(err, ErrInvalidCredentials) {
				t.Fatalf("Login() error = %v, want ErrInvalidCredentials", err)
			}
			messages = append(messages, err.Error())
		})
	}

	if messages[0] != messages[1] {
		t.Fatalf("login errors differ (%q vs %q); they must not reveal whether the email exists", messages[0], messages[1])
	}
}

func TestMeReturnsProfile(t *testing.T) {
	svc, _, _, _ := newTestService()
	user := registerAlice(t, svc)

	id, err := uuid.Parse(user.ID)
	if err != nil {
		t.Fatalf("parsing user ID: %v", err)
	}

	profile, err := svc.Me(context.Background(), id)
	if err != nil {
		t.Fatalf("Me() error = %v", err)
	}
	if profile.Email != "alice@example.com" {
		t.Fatalf("Email = %q, want %q", profile.Email, "alice@example.com")
	}

	_, err = svc.Me(context.Background(), uuid.New())
	if !errors.Is(err, ErrUnauthorized) {
		t.Fatalf("Me(unknown) error = %v, want ErrUnauthorized", err)
	}
}

func TestAuthenticateRejectsRevokedToken(t *testing.T) {
	svc, _, blacklist, manager := newTestService()
	registerAlice(t, svc)

	resp, err := svc.Login(context.Background(), LoginRequest{Email: "alice@example.com", Password: "password123"})
	if err != nil {
		t.Fatalf("Login() error = %v", err)
	}

	claims, err := svc.Authenticate(context.Background(), resp.AccessToken)
	if err != nil {
		t.Fatalf("Authenticate() error = %v", err)
	}

	if err := blacklist.Add(context.Background(), claims.ID, time.Minute); err != nil {
		t.Fatalf("blacklist.Add() error = %v", err)
	}

	if _, err := svc.Authenticate(context.Background(), resp.AccessToken); !errors.Is(err, ErrUnauthorized) {
		t.Fatalf("Authenticate() error = %v, want ErrUnauthorized", err)
	}

	if _, err := manager.ParseAccess(resp.AccessToken); err != nil {
		t.Fatalf("revocation must not tamper with token validity: %v", err)
	}
}

func TestAuthenticateRejectsStaleTokenVersion(t *testing.T) {
	svc, repo, _, _ := newTestService()
	user := registerAlice(t, svc)

	resp, err := svc.Login(context.Background(), LoginRequest{Email: "alice@example.com", Password: "password123"})
	if err != nil {
		t.Fatalf("Login() error = %v", err)
	}

	id, err := uuid.Parse(user.ID)
	if err != nil {
		t.Fatalf("parsing user ID: %v", err)
	}
	if err := repo.IncrementTokenVersion(context.Background(), id); err != nil {
		t.Fatalf("IncrementTokenVersion() error = %v", err)
	}

	if _, err := svc.Authenticate(context.Background(), resp.AccessToken); !errors.Is(err, ErrUnauthorized) {
		t.Fatalf("Authenticate() error = %v, want ErrUnauthorized", err)
	}
}

func TestAuthenticateRejectsMalformedToken(t *testing.T) {
	svc, _, _, _ := newTestService()

	if _, err := svc.Authenticate(context.Background(), "not-a-token"); !errors.Is(err, ErrUnauthorized) {
		t.Fatalf("Authenticate() error = %v, want ErrUnauthorized", err)
	}
}

func loginAlice(t *testing.T, svc *Service) Session {
	t.Helper()
	resp, err := svc.Login(context.Background(), LoginRequest{
		Email:    "alice@example.com",
		Password: "password123",
	})
	if err != nil {
		t.Fatalf("Login() error = %v", err)
	}
	return resp
}

func TestRefreshRotatesTokens(t *testing.T) {
	svc, _, _, manager := newTestService()
	registerAlice(t, svc)
	login := loginAlice(t, svc)

	refreshed, err := svc.Refresh(context.Background(), login.RefreshToken)
	if err != nil {
		t.Fatalf("Refresh() error = %v", err)
	}
	if refreshed.AccessToken == login.AccessToken || refreshed.RefreshToken == login.RefreshToken {
		t.Fatal("Refresh() must issue a new token pair")
	}
	if _, err := svc.Authenticate(context.Background(), refreshed.AccessToken); err != nil {
		t.Fatalf("new access token rejected: %v", err)
	}
	if _, err := manager.ParseRefresh(refreshed.RefreshToken); err != nil {
		t.Fatalf("new refresh token invalid: %v", err)
	}

	if _, err := svc.Refresh(context.Background(), login.RefreshToken); !errors.Is(err, ErrUnauthorized) {
		t.Fatalf("reusing a rotated refresh token: error = %v, want ErrUnauthorized", err)
	}
}

func TestRefreshRejectsAccessToken(t *testing.T) {
	svc, _, _, _ := newTestService()
	registerAlice(t, svc)
	login := loginAlice(t, svc)

	if _, err := svc.Refresh(context.Background(), login.AccessToken); !errors.Is(err, ErrUnauthorized) {
		t.Fatalf("Refresh(access token) error = %v, want ErrUnauthorized", err)
	}
}

func TestRefreshRejectsMissingToken(t *testing.T) {
	svc, _, _, _ := newTestService()

	if _, err := svc.Refresh(context.Background(), ""); !errors.Is(err, ErrUnauthorized) {
		t.Fatalf("Refresh(empty) error = %v, want ErrUnauthorized", err)
	}
}

func TestRefreshRejectsStaleTokenVersion(t *testing.T) {
	svc, repo, _, _ := newTestService()
	user := registerAlice(t, svc)
	login := loginAlice(t, svc)

	id, err := uuid.Parse(user.ID)
	if err != nil {
		t.Fatalf("parsing user ID: %v", err)
	}
	if err := repo.IncrementTokenVersion(context.Background(), id); err != nil {
		t.Fatalf("IncrementTokenVersion() error = %v", err)
	}

	if _, err := svc.Refresh(context.Background(), login.RefreshToken); !errors.Is(err, ErrUnauthorized) {
		t.Fatalf("Refresh() error = %v, want ErrUnauthorized", err)
	}
}

func TestLogoutRevokesSession(t *testing.T) {
	svc, _, _, _ := newTestService()
	registerAlice(t, svc)
	login := loginAlice(t, svc)

	err := svc.Logout(context.Background(), login.AccessToken, login.RefreshToken)
	if err != nil {
		t.Fatalf("Logout() error = %v", err)
	}

	if _, err := svc.Authenticate(context.Background(), login.AccessToken); !errors.Is(err, ErrUnauthorized) {
		t.Fatalf("access token after logout: error = %v, want ErrUnauthorized", err)
	}
	if _, err := svc.Refresh(context.Background(), login.RefreshToken); !errors.Is(err, ErrUnauthorized) {
		t.Fatalf("refresh token after logout: error = %v, want ErrUnauthorized", err)
	}
}

func TestLogoutRejectsMismatchedTokens(t *testing.T) {
	svc, _, _, _ := newTestService()
	registerAlice(t, svc)
	login := loginAlice(t, svc)

	if _, err := svc.Register(context.Background(), RegisterRequest{
		Name:     "Bob",
		Email:    "bob@example.com",
		Password: "password123",
		AppRole:  "user",
	}); err != nil {
		t.Fatalf("Register(Bob) error = %v", err)
	}
	bob, err := svc.Login(context.Background(), LoginRequest{Email: "bob@example.com", Password: "password123"})
	if err != nil {
		t.Fatalf("Login(Bob) error = %v", err)
	}

	err = svc.Logout(context.Background(), login.AccessToken, bob.RefreshToken)
	if !errors.Is(err, ErrUnauthorized) {
		t.Fatalf("Logout(mismatched) error = %v, want ErrUnauthorized", err)
	}
}

func TestLogoutRejectsMissingRefreshToken(t *testing.T) {
	svc, _, _, _ := newTestService()
	registerAlice(t, svc)
	login := loginAlice(t, svc)

	if err := svc.Logout(context.Background(), login.AccessToken, ""); !errors.Is(err, ErrUnauthorized) {
		t.Fatalf("Logout(empty) error = %v, want ErrUnauthorized", err)
	}
}

type memoryUserRepository struct {
	mu      sync.Mutex
	users   map[uuid.UUID]User
	byEmail map[string]uuid.UUID
}

func newMemoryUserRepository() *memoryUserRepository {
	return &memoryUserRepository{
		users:   make(map[uuid.UUID]User),
		byEmail: make(map[string]uuid.UUID),
	}
}

func (r *memoryUserRepository) Create(ctx context.Context, user User) (User, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	if _, exists := r.byEmail[user.Email]; exists {
		return User{}, ErrEmailExists
	}
	now := time.Now().UTC()
	if user.TokenVersion == 0 {
		user.TokenVersion = 1
	}
	user.CreatedAt = now
	user.UpdatedAt = now
	r.users[user.ID] = user
	r.byEmail[user.Email] = user.ID
	return user, nil
}

func (r *memoryUserRepository) GetByEmail(ctx context.Context, email string) (User, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	id, ok := r.byEmail[email]
	if !ok {
		return User{}, ErrNotFound
	}
	return r.users[id], nil
}

func (r *memoryUserRepository) GetByID(ctx context.Context, id uuid.UUID) (User, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	user, ok := r.users[id]
	if !ok {
		return User{}, ErrNotFound
	}
	return user, nil
}

func (r *memoryUserRepository) IncrementTokenVersion(ctx context.Context, id uuid.UUID) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	user, ok := r.users[id]
	if !ok {
		return ErrNotFound
	}
	user.TokenVersion++
	r.users[id] = user
	return nil
}

func (r *memoryUserRepository) count() int {
	r.mu.Lock()
	defer r.mu.Unlock()
	return len(r.users)
}

type memoryTokenBlacklist struct {
	mu      sync.Mutex
	revoked map[string]bool
}

func newMemoryTokenBlacklist() *memoryTokenBlacklist {
	return &memoryTokenBlacklist{revoked: make(map[string]bool)}
}

func (b *memoryTokenBlacklist) Add(ctx context.Context, jti string, ttl time.Duration) error {
	b.mu.Lock()
	defer b.mu.Unlock()
	b.revoked[jti] = true
	return nil
}

func (b *memoryTokenBlacklist) Has(ctx context.Context, jti string) (bool, error) {
	b.mu.Lock()
	defer b.mu.Unlock()
	return b.revoked[jti], nil
}
