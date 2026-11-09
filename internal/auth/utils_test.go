package auth

import (
	"context"
	"testing"
	"time"

	"github.com/alicebob/miniredis/v2"

	kitredis "github.com/YacineMK/SiteTrack/kit/redis"
)

func TestRedisBlacklist(t *testing.T) {
	server := miniredis.RunT(t)
	client, err := kitredis.New(context.Background(), kitredis.Config{Addr: server.Addr()})
	if err != nil {
		t.Fatalf("connecting to miniredis: %v", err)
	}
	t.Cleanup(func() {
		_ = client.Close()
	})

	blacklist := NewRedisBlacklist(client)

	revoked, err := blacklist.Has(context.Background(), "token-1")
	if err != nil {
		t.Fatalf("Has() error = %v", err)
	}
	if revoked {
		t.Fatal("unknown token reported as revoked")
	}

	if err := blacklist.Add(context.Background(), "token-1", 2*time.Second); err != nil {
		t.Fatalf("Add() error = %v", err)
	}
	if !server.Exists("auth:blacklist:token-1") {
		t.Fatal("token not stored under the expected key prefix")
	}

	revoked, err = blacklist.Has(context.Background(), "token-1")
	if err != nil {
		t.Fatalf("Has() error = %v", err)
	}
	if !revoked {
		t.Fatal("blacklisted token reported as valid")
	}

	server.FastForward(3 * time.Second)

	revoked, err = blacklist.Has(context.Background(), "token-1")
	if err != nil {
		t.Fatalf("Has() error = %v", err)
	}
	if revoked {
		t.Fatal("blacklist entry must expire with its TTL")
	}
}
