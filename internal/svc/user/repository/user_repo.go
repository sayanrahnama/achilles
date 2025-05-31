package repository

import (
	"context"
	"database/sql"
	"errors"

	"github.com/hailsayan/achilles/internal/svc/user/model"
)

type UserRepository interface {
	CreateUser(ctx context.Context, user *model.User) error
	GetByUserID(ctx context.Context, id string) (*model.User, error)
	GetByUsername(ctx context.Context, username string) (*model.User, error)
	UpdateUser(ctx context.Context, user *model.User) error
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

func (r *userRepository) CreateUser(ctx context.Context, user *model.User) error {
	query := `
		INSERT INTO
			users (id, username, email, first_name, last_name, created_at, updated_at)
		VALUES
			($1, $2, $3, $4, $5, $6, $7)
	`

	_, err := r.db.ExecContext(ctx, query,
		user.ID,
		user.Username,
		user.Email,
		user.FirstName,
		user.LastName,
		user.CreatedAt,
		user.UpdatedAt,
	)

	return err
}

func (r *userRepository) GetByUserID(ctx context.Context, id string) (*model.User, error) {
	query := `
		SELECT
			id, username, email, first_name, last_name, created_at, updated_at
		FROM
			users
		WHERE
			id = $1
	`

	user := &model.User{}
	err := r.db.QueryRowContext(ctx, query, id).Scan(
		&user.ID,
		&user.Username,
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

func (r *userRepository) GetByUsername(ctx context.Context, username string) (*model.User, error) {
	query := `
		SELECT
			id, username, email, first_name, last_name, created_at, updated_at
		FROM
			users
		WHERE
			username = $1
	`

	user := &model.User{}
	err := r.db.QueryRowContext(ctx, query, username).Scan(
		&user.ID,
		&user.Username,
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

func (r *userRepository) UpdateUser(ctx context.Context, user *model.User) error {
	query := `
		UPDATE
			users
		SET
			username = $1, email = $2, first_name = $3, last_name = $4, updated_at = $5
		WHERE
			id = $6
	`

	_, err := r.db.ExecContext(ctx, query,
		user.Username,
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