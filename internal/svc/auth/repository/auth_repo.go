package repository

import (
	"context"
	"database/sql"
	"errors"

	"github.com/hailsayan/achilles/internal/svc/auth/entity"
)

type AuthRepository interface {
	Create(ctx context.Context, userAuth *entity.UserAuth) error
	GetByID(ctx context.Context, userID string) (*entity.UserAuth, error)
	UpdatePassword(ctx context.Context, userID, hashedPassword string) error
}

type authRepository struct {
	db DBTX
}

func NewAuthRepository(db DBTX) AuthRepository {
	return &authRepository{
		db: db,
	}
}

func (r *authRepository) Create(ctx context.Context, userAuth *entity.UserAuth) error {
	query := `
	INSERT INTO
		user_auth(id, hashed_password)
	VALUES
		($1, $2)
	`

	_, err := r.db.ExecContext(ctx, query, userAuth.ID, userAuth.HashedPassword)
	return err
}

func (r *authRepository) GetByID(ctx context.Context, userID string) (*entity.UserAuth, error) {
	query := `
		SELECT
			id, hashed_password
		FROM
			user_auth
		WHERE
			id = $1
	`

	userAuth := &entity.UserAuth{}
	if err := r.db.QueryRowContext(ctx, query, userID).Scan(&userAuth.ID, &userAuth.HashedPassword); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, nil
		}
		return nil, err
	}

	return userAuth, nil
}

func (r *authRepository) UpdatePassword(ctx context.Context, userID, hashedPassword string) error {
	query := `
		UPDATE
			user_auth
		SET
			hashed_password = $1
		WHERE
			id = $2
	`

	_, err := r.db.ExecContext(ctx, query, hashedPassword, userID)
	return err
}