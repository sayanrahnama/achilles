package repository

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/hailsayan/achilles/internal/svc/user/model"
)

var (
	ErrUserNotFound = errors.New("user not found")
	ErrUserExists   = errors.New("user already exists")
)

type UserRepository interface {
	Create(ctx context.Context, user *model.User) error
	GetByID(ctx context.Context, id string) (*model.User, error)
	GetByUsername(ctx context.Context, username string) (*model.User, error)
	Update(ctx context.Context, user *model.User) error
	Delete(ctx context.Context, id string) error
}

type userRepository struct {
	db DBTX
}

func NewUserRepository(db DBTX) UserRepository {
	return &userRepository{
		db: db,
	}
}

func (r *userRepository) Create(ctx context.Context, user *model.User) error {
	var exists bool
	err := r.db.QueryRowContext(ctx,
		"SELECT EXISTS(SELECT 1 FROM users WHERE username = $1 OR email = $2)",
		strings.ToLower(user.Username), strings.ToLower(user.Email)).Scan(&exists)

	if err != nil {
		return err
	}

	if exists {
		return ErrUserExists
	}

	now := time.Now().UTC()
	user.CreatedAt = now
	user.UpdatedAt = now

	query := `
		INSERT INTO users (id, username, email, first_name, last_name, created_at, updated_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7)
	`

	_, err = r.db.ExecContext(ctx, query,
		user.ID,
		strings.ToLower(user.Username),
		strings.ToLower(user.Email),
		user.FirstName,
		user.LastName,
		user.CreatedAt,
		user.UpdatedAt,
	)

	return err
}

func (r *userRepository) GetByID(ctx context.Context, id string) (*model.User, error) {
	return r.getUserByField(ctx, "id", id)
}

func (r *userRepository) GetByUsername(ctx context.Context, username string) (*model.User, error) {
	return r.getUserByField(ctx, "username", strings.ToLower(username))
}

func (r *userRepository) Update(ctx context.Context, user *model.User) error {
	user.UpdatedAt = time.Now().UTC()

	query := `
		UPDATE users
		SET username = $1, email = $2, first_name = $3, last_name = $4, updated_at = $5
		WHERE id = $6
		RETURNING id
	`

	var id string
	err := r.db.QueryRowContext(ctx, query,
		strings.ToLower(user.Username),
		strings.ToLower(user.Email),
		user.FirstName,
		user.LastName,
		user.UpdatedAt,
		user.ID,
	).Scan(&id)

	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return ErrUserNotFound
		}
		return err
	}

	return nil
}

func (r *userRepository) Delete(ctx context.Context, id string) error {
	query := `
		DELETE FROM users
		WHERE id = $1
		RETURNING id
	`

	var userID string
	err := r.db.QueryRowContext(ctx, query, id).Scan(&userID)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return ErrUserNotFound
		}
		return err
	}

	return nil
}

func (r *userRepository) getUserByField(ctx context.Context, fieldName, fieldValue string) (*model.User, error) {
	user := &model.User{}

	query := fmt.Sprintf(`
		SELECT id, username, email, first_name, last_name, created_at, updated_at
		FROM users
		WHERE %s = $1
	`, fieldName)

	err := r.db.QueryRowContext(ctx, query, fieldValue).Scan(
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
			return nil, ErrUserNotFound
		}
		return nil, err
	}

	return user, nil
}
