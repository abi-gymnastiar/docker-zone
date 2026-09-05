package repository

import (
	"context"

	"dashboard/internal/domain"
)

type ServiceRepository interface {
	List(ctx context.Context) ([]domain.ServiceConfig, error)
	Get(ctx context.Context, name string) (domain.ServiceConfig, error)
	Sync(ctx context.Context, services map[string]domain.ServiceConfig) error
}
