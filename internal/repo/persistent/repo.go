package persistent

import (
	"context"
	"fmt"

	"github.com/cs-parser/pkg/postgres"
)

// Repository предоставляет доступ к постоянному хранилищу.
type Repository struct {
	pg *postgres.Postgres
}

// New создает новый экземпляр репозитория.
func NewRepository(pg *postgres.Postgres) *Repository {
	return &Repository{pg: pg}
}

// Example: метод для получения данных из базы
func (r *Repository) GetTranslation(ctx context.Context, id int) (string, error) {
	var result string
	err := r.pg.Pool.QueryRow(ctx, "SELECT translation FROM translations WHERE id=$1", id).Scan(&result)
	if err != nil {
		return "", fmt.Errorf("GetTranslation failed: %w", err)
	}
	return result, nil
}

// Example: метод для сохранения данных
func (r *Repository) SaveTranslation(ctx context.Context, text string) error {
	_, err := r.pg.Pool.Exec(ctx, "INSERT INTO translations (translation) VALUES ($1)", text)
	if err != nil {
		return fmt.Errorf("SaveTranslation failed: %w", err)
	}
	return nil
}
