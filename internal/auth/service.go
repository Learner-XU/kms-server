package auth

import (
	"database/sql"
	"errors"
	"fmt"
	"time"

	"golang.org/x/crypto/bcrypt"
)

type Service struct {
	db         *sql.DB
	jwtManager *JWTManager
}

func NewService(db *sql.DB, jwtSecret string) (*Service, error) {
	svc := &Service{
		db:         db,
		jwtManager: NewJWTManager(jwtSecret),
	}
	// Ensure refresh_tokens table exists
	if err := svc.migrateRefreshTokens(); err != nil {
		return nil, fmt.Errorf("migrate refresh_tokens: %w", err)
	}
	return svc, nil
}

func (s *Service) migrateRefreshTokens() error {
	query := `CREATE TABLE IF NOT EXISTS refresh_tokens (
		id         BIGINT AUTO_INCREMENT PRIMARY KEY,
		user_id    BIGINT NOT NULL,
		jti        VARCHAR(64) NOT NULL,
		expires_at DATETIME NOT NULL,
		revoked    BOOLEAN NOT NULL DEFAULT FALSE,
		created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
		UNIQUE KEY uq_jti (jti),
		INDEX idx_user_revoked (user_id, revoked)
	) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci`
	if _, err := s.db.Exec(query); err != nil {
		return err
	}
	return nil
}

func (s *Service) Register(req *RegisterRequest) (*User, error) {
	// Check if username or email exists
	var count int
	err := s.db.QueryRow("SELECT COUNT(*) FROM users WHERE username = ? OR email = ?",
		req.Username, req.Email).Scan(&count)
	if err != nil {
		return nil, err
	}
	if count > 0 {
		// H-5: return generic error — don't leak whether user or email was the match
		return nil, errors.New("registration failed")
	}

	// Hash password
	hash, err := bcrypt.GenerateFromPassword([]byte(req.Password), bcrypt.DefaultCost)
	if err != nil {
		return nil, err
	}

	nickname := req.Nickname
	if nickname == "" {
		nickname = req.Username
	}

	result, err := s.db.Exec(
		"INSERT INTO users (username, email, password_hash, nickname, role) VALUES (?, ?, ?, ?, ?)",
		req.Username, req.Email, string(hash), nickname, "member",
	)
	if err != nil {
		return nil, err
	}

	id, _ := result.LastInsertId()
	return &User{
		ID:       id,
		Username: req.Username,
		Email:    req.Email,
		Nickname: nickname,
		Role:     "member",
	}, nil
}

func (s *Service) Login(req *LoginRequest) (*User, error) {
	user := &User{}
	err := s.db.QueryRow(
		"SELECT id, username, email, password_hash, nickname, role, created_at, updated_at FROM users WHERE username = ?",
		req.Username,
	).Scan(&user.ID, &user.Username, &user.Email, &user.PasswordHash, &user.Nickname, &user.Role, &user.CreatedAt, &user.UpdatedAt)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, errors.New("invalid username or password")
		}
		return nil, err
	}

	if err := bcrypt.CompareHashAndPassword([]byte(user.PasswordHash), []byte(req.Password)); err != nil {
		return nil, errors.New("invalid username or password")
	}

	return user, nil
}

func (s *Service) GetUserByID(id int64) (*User, error) {
	user := &User{}
	err := s.db.QueryRow(
		"SELECT id, username, email, password_hash, nickname, role, created_at, updated_at FROM users WHERE id = ?", id,
	).Scan(&user.ID, &user.Username, &user.Email, &user.PasswordHash, &user.Nickname, &user.Role, &user.CreatedAt, &user.UpdatedAt)
	if err != nil {
		return nil, err
	}
	return user, nil
}

// StoreRefreshToken persists the jti for a refresh token so it can be validated
// during rotation.
func (s *Service) StoreRefreshToken(userID int64, jti string) error {
	expires := time.Now().Add(s.jwtManager.RefreshExpiry())
	_, err := s.db.Exec(
		"INSERT INTO refresh_tokens (user_id, jti, expires_at) VALUES (?, ?, ?)",
		userID, jti, expires,
	)
	return err
}

// ValidateAndRotateRefreshToken checks that the jti exists, belongs to the
// user, and has not been revoked. If valid, it revokes the old jti and returns
// true. This implements one-time-use refresh token rotation.
func (s *Service) ValidateAndRotateRefreshToken(userID int64, jti string) (bool, error) {
	// Atomic SELECT + UPDATE in a transaction to prevent TOCTOU race
	tx, err := s.db.Begin()
	if err != nil {
		return false, err
	}
	defer tx.Rollback()

	var dbUserID int64
	var revoked bool
	var expiresAt time.Time

	err = tx.QueryRow(
		"SELECT user_id, revoked, expires_at FROM refresh_tokens WHERE jti = ? FOR UPDATE", jti,
	).Scan(&dbUserID, &revoked, &expiresAt)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return false, nil
		}
		return false, err
	}

	if dbUserID != userID || revoked || time.Now().After(expiresAt) {
		return false, nil
	}

	// Revoke the old token (one-time use)
	result, err := tx.Exec("UPDATE refresh_tokens SET revoked = TRUE WHERE jti = ? AND revoked = FALSE", jti)
	if err != nil {
		return false, err
	}
	rows, _ := result.RowsAffected()
	if rows == 0 {
		// Another concurrent request already revoked it
		return false, nil
	}

	if err := tx.Commit(); err != nil {
		return false, err
	}

	return true, nil
}

// RevokeAllUserRefreshTokens revokes all refresh tokens for a user (e.g. on
// password change or logout-all).
func (s *Service) RevokeAllUserRefreshTokens(userID int64) error {
	_, err := s.db.Exec("UPDATE refresh_tokens SET revoked = TRUE WHERE user_id = ? AND revoked = FALSE", userID)
	return err
}

// CleanupExpiredRefreshTokens removes expired tokens (call periodically).
func (s *Service) CleanupExpiredRefreshTokens() error {
	_, err := s.db.Exec("DELETE FROM refresh_tokens WHERE expires_at < NOW()")
	return err
}

// GenerateTokenPair is a convenience method that creates access + refresh
// tokens and stores the refresh jti.
func (s *Service) GenerateTokenPair(user *User) (*TokenResponse, error) {
	access, err := s.jwtManager.GenerateAccessToken(user)
	if err != nil {
		return nil, err
	}
	refresh, jti, err := s.jwtManager.GenerateRefreshToken(user)
	if err != nil {
		return nil, err
	}
	if err := s.StoreRefreshToken(user.ID, jti); err != nil {
		return nil, err
	}

	return &TokenResponse{
		AccessToken:  access,
		RefreshToken: refresh,
		ExpiresIn:    s.jwtManager.AccessExpirySeconds(),
		User:         *user,
	}, nil
}

func (s *Service) JWTManager() *JWTManager {
	return s.jwtManager
}


