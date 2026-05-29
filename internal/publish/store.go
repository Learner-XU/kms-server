package publish

import (
	"database/sql"
	"encoding/json"
	"fmt"
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
		CREATE TABLE IF NOT EXISTS published (
			slug         VARCHAR(200) PRIMARY KEY,
			note_path    VARCHAR(500) NOT NULL UNIQUE,
			username     VARCHAR(100) NOT NULL,
			nickname     VARCHAR(200) NOT NULL DEFAULT '',
			title        VARCHAR(500) NOT NULL,
			summary      TEXT,
			tags         JSON,
			published_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
			INDEX idx_published_user (username),
			INDEX idx_published_at (published_at)
		) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci
	`)
	if err != nil {
		return fmt.Errorf("create published table: %w", err)
	}
	return nil
}

func (s *Store) Publish(slug, notePath, username, nickname, title, summary string, tags []string) error {
	tagsJSON, _ := json.Marshal(tags)
	_, err := s.db.Exec(`
		INSERT INTO published (slug, note_path, username, nickname, title, summary, tags, published_at)
		VALUES (?, ?, ?, ?, ?, ?, ?, NOW())
		ON DUPLICATE KEY UPDATE slug=VALUES(slug), title=VALUES(title), summary=VALUES(summary), tags=VALUES(tags)
	`, slug, notePath, username, nickname, title, summary, string(tagsJSON))
	return err
}

func (s *Store) Unpublish(notePath string) error {
	_, err := s.db.Exec("DELETE FROM published WHERE note_path = ?", notePath)
	return err
}

func (s *Store) GetBySlug(slug string) (*Published, error) {
	var p Published
	var tagsJSON string
	err := s.db.QueryRow(
		"SELECT slug, note_path, username, nickname, title, summary, tags, published_at FROM published WHERE slug = ?", slug,
	).Scan(&p.Slug, &p.NotePath, &p.Username, &p.Nickname, &p.Title, &p.Summary, &tagsJSON, &p.PublishedAt)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, nil
		}
		return nil, err
	}
	json.Unmarshal([]byte(tagsJSON), &p.Tags)
	if p.Tags == nil {
		p.Tags = []string{}
	}
	return &p, nil
}

func (s *Store) GetByNotePath(notePath string) (*Published, error) {
	var p Published
	var tagsJSON string
	err := s.db.QueryRow(
		"SELECT slug, note_path, username, nickname, title, summary, tags, published_at FROM published WHERE note_path = ?", notePath,
	).Scan(&p.Slug, &p.NotePath, &p.Username, &p.Nickname, &p.Title, &p.Summary, &tagsJSON, &p.PublishedAt)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, nil
		}
		return nil, err
	}
	json.Unmarshal([]byte(tagsJSON), &p.Tags)
	if p.Tags == nil {
		p.Tags = []string{}
	}
	return &p, nil
}

func (s *Store) List(limit, offset int) ([]Published, int, error) {
	var total int
	s.db.QueryRow("SELECT COUNT(*) FROM published").Scan(&total)

	rows, err := s.db.Query(
		"SELECT slug, note_path, username, nickname, title, summary, tags, published_at FROM published ORDER BY published_at DESC LIMIT ? OFFSET ?",
		limit, offset,
	)
	if err != nil {
		return nil, 0, err
	}
	defer rows.Close()

	var items []Published
	for rows.Next() {
		var p Published
		var tagsJSON string
		if err := rows.Scan(&p.Slug, &p.NotePath, &p.Username, &p.Nickname, &p.Title, &p.Summary, &tagsJSON, &p.PublishedAt); err != nil {
			continue
		}
		json.Unmarshal([]byte(tagsJSON), &p.Tags)
		if p.Tags == nil {
			p.Tags = []string{}
		}
		items = append(items, p)
	}
	if items == nil {
		items = make([]Published, 0)
	}
	return items, total, nil
}

func (s *Store) IsSlugTaken(slug string) (bool, error) {
	var count int
	err := s.db.QueryRow("SELECT COUNT(*) FROM published WHERE slug = ?", slug).Scan(&count)
	return count > 0, err
}
