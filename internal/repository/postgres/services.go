package postgres

import (
	"context"
	"fmt"

	"dashboard/internal/domain"
	"github.com/jackc/pgx/v5/pgxpool"
)

type Services struct {
	pool *pgxpool.Pool
}

func NewServices(pool *pgxpool.Pool) *Services {
	return &Services{pool: pool}
}

func (r *Services) List(ctx context.Context) ([]domain.ServiceConfig, error) {
	rows, err := r.pool.Query(ctx, `
		SELECT s.name, s.container, s.description, COALESCE(array_agg(a.action ORDER BY a.action)
		FILTER (WHERE a.action IS NOT NULL), '{}')
		FROM services s
		LEFT JOIN service_actions a ON a.service_name = s.name
		GROUP BY s.name, s.container, s.description
		ORDER BY s.name`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var services []domain.ServiceConfig
	for rows.Next() {
		var service domain.ServiceConfig
		if err := rows.Scan(&service.Name, &service.Container, &service.Description, &service.Actions); err != nil {
			return nil, err
		}
		services = append(services, service)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	return services, nil
}

func (r *Services) Get(ctx context.Context, name string) (domain.ServiceConfig, error) {
	var service domain.ServiceConfig
	err := r.pool.QueryRow(ctx, `
		SELECT s.name, s.container, s.description, COALESCE(array_agg(a.action ORDER BY a.action)
		FILTER (WHERE a.action IS NOT NULL), '{}')
		FROM services s
		LEFT JOIN service_actions a ON a.service_name = s.name
		WHERE s.name = $1
		GROUP BY s.name, s.container, s.description`, name).
		Scan(&service.Name, &service.Container, &service.Description, &service.Actions)
	if err != nil {
		return domain.ServiceConfig{}, err
	}
	return service, nil
}

func (r *Services) Sync(ctx context.Context, configs map[string]domain.ServiceConfig) error {
	tx, err := r.pool.Begin(ctx)
	if err != nil {
		return err
	}
	defer tx.Rollback(ctx)

	for _, service := range configs {
		_, err = tx.Exec(ctx, `
			INSERT INTO services (name, container, description) VALUES ($1, $2, $3)
			ON CONFLICT (name) DO UPDATE SET container = EXCLUDED.container, description = EXCLUDED.description`,
			service.Name, service.Container, service.Description)
		if err != nil {
			return err
		}
		if _, err = tx.Exec(ctx, `DELETE FROM service_actions WHERE service_name = $1`, service.Name); err != nil {
			return err
		}
		for _, action := range service.Actions {
			if _, err = tx.Exec(ctx, `INSERT INTO service_actions (service_name, action) VALUES ($1, $2)`, service.Name, action); err != nil {
				return err
			}
		}
	}
	return tx.Commit(ctx)
}

func RunMigrations(ctx context.Context, pool *pgxpool.Pool, migration string) error {
	if _, err := pool.Exec(ctx, migration); err != nil {
		return fmt.Errorf("run migrations: %w", err)
	}
	return nil
}
