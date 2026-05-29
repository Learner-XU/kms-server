package profile

import (
	"database/sql"
	"errors"
	"encoding/json"
	"fmt"
	"time"
)

type Store struct {
	db *sql.DB
}

func NewStore(db *sql.DB) (*Store, error) {
	s := &Store{db: db}
	if err := s.migrate(); err != nil {
		return nil, err
	}
	return s, nil
}

func (s *Store) migrate() error {
	_, err := s.db.Exec(`
		CREATE TABLE IF NOT EXISTS profiles (
			username VARCHAR(100) PRIMARY KEY,
			data     JSON NOT NULL,
			updated_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP
		) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci
	`)
	if err != nil {
		return fmt.Errorf("create profiles table: %w", err)
	}
	return nil
}

func (s *Store) Get(username string) (*Profile, error) {
	var dataJSON string
	var updatedAt time.Time
	err := s.db.QueryRow("SELECT data, updated_at FROM profiles WHERE username = ?", username).
		Scan(&dataJSON, &updatedAt)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, nil // not found — return empty profile
		}
		return nil, err
	}
	var p Profile
	if err := json.Unmarshal([]byte(dataJSON), &p); err != nil {
		return nil, err
	}
	p.Username = username
	p.UpdatedAt = updatedAt
	return &p, nil
}

func (s *Store) Upsert(username string, p *Profile) error {
	data, err := json.Marshal(p)
	if err != nil {
		return err
	}
	_, err = s.db.Exec(`
		INSERT INTO profiles (username, data, updated_at) VALUES (?, ?, NOW())
		ON DUPLICATE KEY UPDATE data = VALUES(data), updated_at = NOW()
	`, username, string(data))
	return err
}

func (s *Store) List() ([]Profile, error) {
	rows, err := s.db.Query("SELECT username, data, updated_at FROM profiles ORDER BY updated_at DESC")
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var profiles []Profile
	for rows.Next() {
		var dataJSON string
		var p Profile
		if err := rows.Scan(&p.Username, &dataJSON, &p.UpdatedAt); err != nil {
			return nil, err
		}
		if err := json.Unmarshal([]byte(dataJSON), &p); err != nil {
			continue
		}
		profiles = append(profiles, p)
	}
	if profiles == nil {
		profiles = make([]Profile, 0)
	}
	return profiles, rows.Err()
}
