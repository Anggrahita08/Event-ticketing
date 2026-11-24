package repository

import (
	"context"
	"database/sql"
	"project-ticketing-golang/internal/model"
)

type UserRepository interface {
	Create(ctx context.Context, user *model.User) error
	FindByEmail(ctx context.Context, email string) (*model.User, error)
	FindByID(ctx context.Context, id string) (*model.User, error)
}

type userRepository struct {
	db *sql.DB
}

func NewUserRepository(db *sql.DB) UserRepository {
	return &userRepository{db: db}
}

func (r *userRepository) Create(ctx context.Context, user *model.User) error {
	query := `INSERT INTO users (name, email, password_hash, role) VALUES ($1, $2, $3, $4) RETURNING id, created_at`
	return r.db.QueryRowContext(ctx, query, user.Name, user.Email, user.PasswordHash, user.Role).Scan(&user.ID, &user.CreatedAt)
}

func (r *userRepository) FindByEmail(ctx context.Context, email string) (*model.User, error) {
	query := `SELECT id, name, email, password_hash, role, created_at FROM users WHERE LOWER(email) = LOWER($1)`
	row := r.db.QueryRowContext(ctx, query, email)
	var u model.User
	err := row.Scan(&u.ID, &u.Name, &u.Email, &u.PasswordHash, &u.Role, &u.CreatedAt)
	if err != nil {
		return nil, err
	}
	return &u, nil
}

func (r *userRepository) FindByID(ctx context.Context, id string) (*model.User, error) {
	query := `SELECT id, name, email, password_hash, role, created_at FROM users WHERE id = $1`
	row := r.db.QueryRowContext(ctx, query, id)
	var u model.User
	err := row.Scan(&u.ID, &u.Name, &u.Email, &u.PasswordHash, &u.Role, &u.CreatedAt)
	if err != nil {
		return nil, err
	}
	return &u, nil
}
