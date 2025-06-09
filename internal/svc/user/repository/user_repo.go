package repository

import (
	"context"
	"database/sql"
	"errors"

	"github.com/hailsayan/achilles/internal/svc/user/entity"
)

type UserRepository interface {
	CreateUser(ctx context.Context, user *entity.User) error
	GetByUserID(ctx context.Context, id string) (*entity.User, error)
	GetByEmail(ctx context.Context, email string) (*entity.User, error)
	UpdateUser(ctx context.Context, user *entity.User) error
	DeleteUserByID(ctx context.Context, id string) error
}

type userRepository struct {
	db DBTX
}

func NewUserRepository(db DBTX) UserRepository {
	return &userRepository{
		db: db,
	}
}

func (r *userRepository) CreateUser(ctx context.Context, user *entity.User) error {
	query := `
		INSERT INTO
			users (id, email, first_name, last_name, created_at, updated_at)
		VALUES
			($1, $2, $3, $4, $5, $6)
	`

	_, err := r.db.ExecContext(ctx, query,
		user.ID,
		user.Email,
		user.FirstName,
		user.LastName,
		user.CreatedAt,
		user.UpdatedAt,
	)

	return err
}

func (r *userRepository) GetByUserID(ctx context.Context, id string) (*entity.User, error) {
	query := `
		SELECT
			id, email, first_name, last_name, created_at, updated_at
		FROM
			users
		WHERE
			id = $1
	`

	user := &entity.User{}
	err := r.db.QueryRowContext(ctx, query, id).Scan(
		&user.ID,
		&user.Email,
		&user.FirstName,
		&user.LastName,
		&user.CreatedAt,
		&user.UpdatedAt,
	)

	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, nil
		}
		return nil, err
	}

	return user, nil
}

func (r *userRepository) GetByEmail(ctx context.Context, email string) (*entity.User, error) {
	query := `
		SELECT
			id, email, first_name, last_name, created_at, updated_at
		FROM
			users
		WHERE
			email = $1
	`

	user := &entity.User{}
	err := r.db.QueryRowContext(ctx, query, email).Scan(
		&user.ID,
		&user.Email,
		&user.FirstName,
		&user.LastName,
		&user.CreatedAt,
		&user.UpdatedAt,
	)

	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, nil
		}
		return nil, err
	}

	return user, nil
}

func (r *userRepository) UpdateUser(ctx context.Context, user *entity.User) error {
	query := `
		UPDATE
			users
		SET
			email = $1, first_name = $2, last_name = $3, updated_at = $4
		WHERE
			id = $5
	`

	_, err := r.db.ExecContext(ctx, query,
		user.Email,
		user.FirstName,
		user.LastName,
		user.UpdatedAt,
		user.ID,
	)

	return err
}

func (r *userRepository) DeleteUserByID(ctx context.Context, id string) error {
	query := `
		DELETE FROM
			users
		WHERE
			id = $1
	`

	_, err := r.db.ExecContext(ctx, query, id)
	return err
}