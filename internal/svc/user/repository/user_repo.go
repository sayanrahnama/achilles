package repository

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"time"

	"github.com/hailsayan/achilles/internal/pkg/logger"
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
	GetUsers(ctx context.Context, page, pageSize int, sortBy string, sortDesc bool) ([]*model.User, int, error)
	Update(ctx context.Context, user *model.User) error
	Delete(ctx context.Context, id string) error
}

type userRepository struct {
	db     DBTX
	cache  CacheRepository
	logger logger.Logger
}

func NewUserRepository(db DBTX, cache CacheRepository, logger logger.Logger) UserRepository {
	return &userRepository{
		db:     db,
		cache:  cache,
		logger: logger,
	}
}

func (r *userRepository) Create(ctx context.Context, user *model.User) error {
	var exists bool
    err := r.db.QueryRowContext(ctx,
        "SELECT EXISTS(SELECT 1 FROM users WHERE username = $1 OR email = $2)",
        user.Username, user.Email).Scan(&exists)

    if err != nil {
        return err
    }

    if exists {
        return ErrUserExists
    }

	now := time.Now()
	user.CreatedAt = now
	user.UpdatedAt = now

	query := `
		INSERT INTO users (id, username, email, first_name, last_name, created_at, updated_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7)
	`

	_, err = r.db.ExecContext(ctx, query,
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

func (r *userRepository) GetByID(ctx context.Context, id string) (*model.User, error) {
	if r.cache != nil {
		cacheKey := r.cache.UserIDKey(id)

		// Try to get from cache first
		user, err := r.cache.GetUser(ctx, cacheKey)
		if err == nil {
			return user, nil
		}

		// If not in cache, get from database
		user, err = r.getUserByField(ctx, "id", id)
		if err != nil {
			return nil, err
		}

		// Store in cache for future requests
		_ = r.cache.SetUser(ctx, cacheKey, user)

		return user, nil
	}

	return r.getUserByField(ctx, "id", id)
}

func (r *userRepository) GetByUsername(ctx context.Context, username string) (*model.User, error) {
	if r.cache != nil {
		cacheKey := r.cache.UsernameKey(username)

		// Try to get from cache first
		user, err := r.cache.GetUser(ctx, cacheKey)
		if err == nil {
			return user, nil
		}

		// If not in cache, get from database
		user, err = r.getUserByField(ctx, "username", username)
		if err != nil {
			return nil, err
		}

		// Store in cache for future requests
		_ = r.cache.SetUser(ctx, cacheKey, user)

		return user, nil
	}

	return r.getUserByField(ctx, "username", username)
}

func (r *userRepository) GetUsers(ctx context.Context, page, pageSize int, sortBy string, sortDesc bool) ([]*model.User, int, error) {
	page, pageSize = r.validatePagination(page, pageSize)
	offset := (page - 1) * pageSize

	sortField, sortOrder := r.validateAndGetSortOptions(sortBy, sortDesc)

	var total int
	err := r.db.QueryRowContext(ctx, "SELECT COUNT(*) FROM users").Scan(&total)
	if err != nil {
		return nil, 0, err
	}

	query := fmt.Sprintf(`
		SELECT id, username, email, first_name, last_name, created_at, updated_at
		FROM users
		ORDER BY %s %s
		LIMIT $1 OFFSET $2
	`, sortField, sortOrder)

	rows, err := r.db.QueryContext(ctx, query, pageSize, offset)
	if err != nil {
		return nil, 0, err
	}
	defer rows.Close()

	users, err := r.scanUserRows(rows)
	if err != nil {
		return nil, 0, err
	}

	return users, total, nil
}

func (r *userRepository) Update(ctx context.Context, user *model.User) error {
	user.UpdatedAt = time.Now()

	query := `
		UPDATE users
		SET email = $1, first_name = $2, last_name = $3, updated_at = $4
		WHERE id = $5
		RETURNING id
	`

	var id string
	err := r.db.QueryRowContext(ctx, query,
		user.Email,
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

	// Invalidate cache
	if r.cache != nil {
		_ = r.invalidateUserCache(ctx, user.ID, user.Username)
	}

	return nil
}

func (r *userRepository) Delete(ctx context.Context, id string) error {
	query := `
		DELETE FROM users
		WHERE id = $1
		RETURNING username
	`

	var username string
	err := r.db.QueryRowContext(ctx, query, id).Scan(&username)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return ErrUserNotFound
		}
		return err
	}

	// Invalidate cache
	if r.cache != nil {
		_ = r.invalidateUserCache(ctx, id, username)
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

func (r *userRepository) validatePagination(page, pageSize int) (int, int) {
	if page < 1 {
		page = 1
	}
	if pageSize < 1 {
		pageSize = 10
	}
	return page, pageSize
}

func (r *userRepository) validateAndGetSortOptions(sortBy string, sortDesc bool) (string, string) {
	allowedSortFields := map[string]string{
		"username":   "username",
		"email":      "email",
		"created_at": "created_at",
		"updated_at": "updated_at",
	}

	sortField, ok := allowedSortFields[sortBy]
	if !ok {
		sortField = "created_at"
	}

	sortOrder := "ASC"
	if sortDesc {
		sortOrder = "DESC"
	}

	return sortField, sortOrder
}

func (r *userRepository) scanUserRows(rows *sql.Rows) ([]*model.User, error) {
	var users []*model.User

	for rows.Next() {
		var user model.User

		err := rows.Scan(
			&user.ID,
			&user.Username,
			&user.Email,
			&user.FirstName,
			&user.LastName,
			&user.CreatedAt,
			&user.UpdatedAt,
		)

		if err != nil {
			return nil, err
		}

		users = append(users, &user)
	}

	if err := rows.Err(); err != nil {
		return nil, err
	}

	return users, nil
}

func (r *userRepository) invalidateUserCache(ctx context.Context, id, username string) error {
	if r.cache == nil {
		return nil
	}
	
	if err := r.cache.DeleteUser(ctx, r.cache.UserIDKey(id)); err != nil {
		return err
	}

	if username != "" {
		if err := r.cache.DeleteUser(ctx, r.cache.UsernameKey(username)); err != nil {
			return err
		}
	}
	
	return nil
}