package auth

import (
	"github.com/jackc/pgx/v5/pgxpool"

	kitjwt "github.com/YacineMK/SiteTrack/kit/jwt"
	kitredis "github.com/YacineMK/SiteTrack/kit/redis"
)

type Dependencies struct {
	DB           *pgxpool.Pool
	Redis        *kitredis.Redis
	JWT          *kitjwt.Manager
	CookieSecure bool
}

func New(deps Dependencies) *Handler {
	repo := NewPostgresRepository(deps.DB)
	svc := NewService(repo, deps.JWT, NewRedisBlacklist(deps.Redis))
	return NewHandler(svc, deps.CookieSecure)
}
