package repository

import (
	"database/sql"
	"fmt"

	"mangahub/pkg/models"
)

type AuthRepository interface {
	CreateUser(username, email, passwordHash string) (int64, error)
	FindByUsername(username string) (*models.User, error)
	FindByEmail(email string) (*models.User, error)
}

type authRepositoryImpl struct {
	db *sql.DB
}

func NewAuthRepository(db *sql.DB) AuthRepository {
	return &authRepositoryImpl{db: db}
}

func (r *authRepositoryImpl) CreateUser(username, email, passwordHash string) (int64, error) {
	result, err := r.db.Exec(
		"INSERT INTO users (username, email, password_hash) VALUES (?, ?, ?)",
		username, email, passwordHash,
	)
	if err != nil {
		return 0, fmt.Errorf("insert user: %w", err)
	}

	id, err := result.LastInsertId()
	if err != nil {
		return 0, fmt.Errorf("get last insert id: %w", err)
	}

	return id, nil
}

func (r *authRepositoryImpl) FindByUsername(username string) (*models.User, error) {
	var user models.User
	err := r.db.QueryRow(
		"SELECT id, username, email, password_hash FROM users WHERE username = ?",
		username,
	).Scan(&user.ID, &user.Username, &user.Email, &user.PasswordHash)

	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("query user by username: %w", err)
	}

	return &user, nil
}

func (r *authRepositoryImpl) FindByEmail(email string) (*models.User, error) {
	var user models.User
	err := r.db.QueryRow(
		"SELECT id, username, email, password_hash FROM users WHERE email = ?",
		email,
	).Scan(&user.ID, &user.Username, &user.Email, &user.PasswordHash)

	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("query user by email: %w", err)
	}

	return &user, nil
}
