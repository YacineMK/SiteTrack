package auth

import (
	"context"
	"sync"
	"time"

	"github.com/google/uuid"
)

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
