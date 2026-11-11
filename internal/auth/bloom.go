package auth

import (
	"context"
	"sync"

	"github.com/bits-and-blooms/bloom/v3"
)

type EmailBloom struct {
	mu     sync.RWMutex
	filter *bloom.BloomFilter
}

func NewEmailBloom() *EmailBloom {
	return &EmailBloom{filter: bloom.NewWithEstimates(100_000, 0.01)}
}

func (b *EmailBloom) Warm(ctx context.Context, repo Repository) error {
	emails, err := repo.ListAllEmails(ctx)
	if err != nil {
		return err
	}
	b.mu.Lock()
	defer b.mu.Unlock()
	for _, e := range emails {
		b.filter.AddString(e)
	}
	return nil
}

func (b *EmailBloom) Add(email string) {
	b.mu.Lock()
	defer b.mu.Unlock()
	b.filter.AddString(email)
}

func (b *EmailBloom) MaybeExists(email string) bool {
	b.mu.RLock()
	defer b.mu.RUnlock()
	return b.filter.TestString(email)
}
