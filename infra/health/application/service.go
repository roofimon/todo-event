package application

import (
	"context"

	"github.com/samber/mo"
	"todoe/infra/health/domain"
	"todoe/infra/health/port"
)

type Service struct {
	repo port.Repository
}

var _ port.UseCase = (*Service)(nil)

func NewService(repo port.Repository) *Service {
	return &Service{repo: repo}
}

func (s *Service) CheckHealth(ctx context.Context) mo.Result[domain.Health] {
	if s.repo.Ping(ctx).IsError() {
		return mo.Ok(domain.Health{Status: "error", Mongo: "unreachable"})
	}
	return mo.Ok(domain.Health{Status: "ok", Mongo: "ok"})
}
