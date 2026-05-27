package auth

import (
	"database/sql"
	"errors"

	"golang.org/x/crypto/bcrypt"
)

type Service struct {
	db         *sql.DB
	jwtManager *JWTManager
}

func NewService(db *sql.DB, jwtSecret string) *Service {
	return &Service{
		db:         db,
		jwtManager: NewJWTManager(jwtSecret),
	}
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
		return nil, errors.New("username or email already exists")
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

func (s *Service) JWTManager() *JWTManager {
	return s.jwtManager
}
