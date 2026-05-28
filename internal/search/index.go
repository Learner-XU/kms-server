package search

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	_ "github.com/go-sql-driver/mysql"
	"github.com/rs/zerolog/log"
)

type Indexer struct {
	db *sql.DB
}

type IndexedNote struct {
	ID       string
	Path     string
	Title    string
	Content  string
	Type     string
	Status   string
	Tags     []string
	Summary  string
	Source   string
	SHA      string
	Created  time.Time
	Updated  time.Time
	Links    []LinkRef
}

type LinkRef struct {
	TargetID    string
	TargetTitle string
	Context     string
}

// NewIndexedNote creates an IndexedNote from individual fields.
// Centralizes the Note→IndexedNote mapping to avoid duplication.
func NewIndexedNote(id, path, title, content, noteType, status string, tags []string, summary, source, sha string, created, updated time.Time) *IndexedNote {
	return &IndexedNote{
		ID: id, Path: path, Title: title, Content: content,
		Type: noteType, Status: status, Tags: tags,
		Summary: summary, Source: source, SHA: sha,
		Created: created, Updated: updated,
	}
}

type SearchResult struct {
	ID      string   `json:"id"`
	Title   string   `json:"title"`
	Path    string   `json:"path"`
	Type    string   `json:"type"`
	Status  string   `json:"status"`
	Tags    []string `json:"tags"`
	Summary string   `json:"summary"`
	Snippet string   `json:"snippet"`
	Score   float64  `json:"score"`
}

type SearchFilters struct {
	Type   string
	Status string
	Tags   []string
}

// NewIndexerWithDB creates an Indexer with an existing database connection.
func NewIndexerWithDB(db *sql.DB) (*Indexer, error) {
	if err := db.Ping(); err != nil {
		return nil, fmt.Errorf("ping mysql: %w", err)
	}
	if err := migrate(db); err != nil {
		return nil, err
	}
	return &Indexer{db: db}, nil
}

func migrate(db *sql.DB) error {
	_, err := db.Exec(`
		CREATE TABLE IF NOT EXISTS notes (
			id VARCHAR(26) PRIMARY KEY,
			path VARCHAR(500) NOT NULL UNIQUE,
			title VARCHAR(500) NOT NULL,
			content LONGTEXT NOT NULL,
			type VARCHAR(20) NOT NULL DEFAULT 'note',
			status VARCHAR(20) NOT NULL DEFAULT 'seed',
			tags TEXT,
			summary TEXT,
			source VARCHAR(500),
			sha VARCHAR(40),
			created DATETIME,
			updated DATETIME,
			INDEX idx_notes_type (type),
			INDEX idx_notes_status (status),
			FULLTEXT INDEX ft_notes (title, content, tags, summary) WITH PARSER ngram
		) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci
	`)
	if err != nil {
		return fmt.Errorf("create notes table: %w", err)
	}

	_, err = db.Exec(`
		CREATE TABLE IF NOT EXISTS links (
			source_id VARCHAR(26) NOT NULL,
			target_id VARCHAR(26) NOT NULL,
			target_title VARCHAR(500),
			context TEXT,
			created DATETIME DEFAULT CURRENT_TIMESTAMP,
			PRIMARY KEY (source_id, target_id),
			INDEX idx_links_target (target_id),
			INDEX idx_links_source (source_id)
		) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci
	`)
	if err != nil {
		return fmt.Errorf("create links table: %w", err)
	}

	return nil
}

func (idx *Indexer) UpsertNote(note *IndexedNote) error {
	tagsJSON, _ := json.Marshal(note.Tags)

	tx, err := idx.db.Begin()
	if err != nil {
		return err
	}
	defer tx.Rollback()

	_, err = tx.Exec(`
		INSERT INTO notes (id, path, title, content, type, status, tags, summary, source, sha, created, updated)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
		ON DUPLICATE KEY UPDATE
			path=VALUES(path), title=VALUES(title), content=VALUES(content),
			type=VALUES(type), status=VALUES(status), tags=VALUES(tags),
			summary=VALUES(summary), source=VALUES(source), sha=VALUES(sha),
			updated=VALUES(updated)
	`, note.ID, note.Path, note.Title, note.Content, note.Type, note.Status,
		string(tagsJSON), note.Summary, note.Source, note.SHA, note.Created, note.Updated)
	if err != nil {
		return err
	}

	_, err = tx.Exec(`DELETE FROM links WHERE source_id = ?`, note.ID)
	if err != nil {
		return err
	}

	for _, link := range note.Links {
		_, err = tx.Exec(`
			INSERT IGNORE INTO links (source_id, target_id, target_title, context)
			VALUES (?, ?, ?, ?)
		`, note.ID, link.TargetID, link.TargetTitle, link.Context)
		if err != nil {
			return err
		}
	}

	return tx.Commit()
}

func (idx *Indexer) DeleteNote(path string) error {
	tx, err := idx.db.Begin()
	if err != nil {
		return err
	}
	defer tx.Rollback()

	// Get note ID first to clean up links
	var noteID string
	err = tx.QueryRow(`SELECT id FROM notes WHERE path = ?`, path).Scan(&noteID)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil // already gone from index
		}
		return err
	}

	// Delete links referencing this note
	_, _ = tx.Exec(`DELETE FROM links WHERE source_id = ?`, noteID)
	_, _ = tx.Exec(`DELETE FROM links WHERE target_id = ?`, noteID)

	// Delete the note itself
	_, err = tx.Exec(`DELETE FROM notes WHERE id = ?`, noteID)
	if err != nil {
		return err
	}

	return tx.Commit()
}

func (idx *Indexer) Search(query string, filters SearchFilters, limit, offset int) ([]SearchResult, int, error) {
	q := strings.TrimSpace(query)
	if q == "" {
		return nil, 0, nil
	}

	// Build LIKE conditions for each word
	words := strings.Fields(q)
	var likeParts []string
	var likeArgs []interface{}
	for _, w := range words {
		likeParts = append(likeParts, "(title LIKE ? OR content LIKE ? OR tags LIKE ? OR summary LIKE ?)")
		pat := "%" + w + "%"
		likeArgs = append(likeArgs, pat, pat, pat, pat)
	}
	where := []string{strings.Join(likeParts, " AND ")}
	args := make([]interface{}, len(likeArgs))
	copy(args, likeArgs)

	if filters.Type != "" {
		where = append(where, "type = ?")
		args = append(args, filters.Type)
	}
	if filters.Status != "" {
		where = append(where, "status = ?")
		args = append(args, filters.Status)
	}
	for _, tag := range filters.Tags {
		where = append(where, "tags LIKE ?")
		args = append(args, "%"+tag+"%")
	}

	whereClause := strings.Join(where, " AND ")

	var total int
	countQuery := "SELECT COUNT(*) FROM notes WHERE " + whereClause
	if err := idx.db.QueryRow(countQuery, args...).Scan(&total); err != nil {
		return nil, 0, err
	}

	searchQuery := fmt.Sprintf(`
		SELECT id, path, title, type, status, tags, summary,
			   SUBSTRING(content, 1, 200) as snippet, 1.0 as score
		FROM notes
		WHERE %s
		ORDER BY updated DESC
		LIMIT ? OFFSET ?
	`, whereClause)

	args = append(args, limit, offset)

	rows, err := idx.db.Query(searchQuery, args...)
	if err != nil {
		return nil, 0, err
	}
	defer rows.Close()

	results := make([]SearchResult, 0)
	for rows.Next() {
		var r SearchResult
		var tagsJSON string
		if err := rows.Scan(&r.ID, &r.Path, &r.Title, &r.Type, &r.Status,
			&tagsJSON, &r.Summary, &r.Snippet, &r.Score); err != nil {
			return nil, 0, err
		}
		json.Unmarshal([]byte(tagsJSON), &r.Tags)
		if r.Tags == nil {
			r.Tags = []string{}
		}
		results = append(results, r)
	}
	if err := rows.Err(); err != nil {
		return nil, 0, err
	}
	return results, total, nil
}

func (idx *Indexer) GetBacklinks(noteID string) ([]SearchResult, error) {
	rows, err := idx.db.Query(`
		SELECT n.id, n.path, n.title, n.type, n.status, n.tags, n.summary
		FROM links l
		JOIN notes n ON n.id = l.source_id
		WHERE l.target_id = ?
		ORDER BY n.updated DESC
	`, noteID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	results := make([]SearchResult, 0)
	for rows.Next() {
		var r SearchResult
		var tagsJSON string
		if err := rows.Scan(&r.ID, &r.Path, &r.Title, &r.Type, &r.Status, &tagsJSON, &r.Summary); err != nil {
			return nil, err
		}
		json.Unmarshal([]byte(tagsJSON), &r.Tags)
		if r.Tags == nil {
			r.Tags = []string{}
		}
		results = append(results, r)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	return results, nil
}

// RawDB returns the underlying database connection.
// Deprecated: prefer adding methods to Indexer instead of direct DB access.
// Currently only used by graph.Builder which queries notes+links jointly.
func (idx *Indexer) RawDB() *sql.DB {
	return idx.db
}

// InsertLink inserts a link edge between two notes.
func (idx *Indexer) InsertLink(sourceID, targetID, targetTitle string) error {
	_, err := idx.db.Exec(`INSERT IGNORE INTO links (source_id, target_id, target_title, context) VALUES (?, ?, ?, '')`,
		sourceID, targetID, targetTitle)
	return err
}

// ListByDir returns all indexed notes under a directory path.
// If dirPath is empty, returns all notes.
func (idx *Indexer) ListByDir(dirPath string) ([]*IndexedNote, error) {
	query := "SELECT id, path, title, content, type, status, tags, summary, source, sha, created, updated FROM notes"
	var args []interface{}
	if dirPath != "" {
		query += " WHERE path LIKE ?"
		args = append(args, dirPath+"/%")
	}
	query += " ORDER BY updated DESC"

	rows, err := idx.db.Query(query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	notes := make([]*IndexedNote, 0)
	for rows.Next() {
		var n IndexedNote
		var tagsJSON string
		if err := rows.Scan(&n.ID, &n.Path, &n.Title, &n.Content, &n.Type, &n.Status,
			&tagsJSON, &n.Summary, &n.Source, &n.SHA, &n.Created, &n.Updated); err != nil {
			return nil, err
		}
		if err := json.Unmarshal([]byte(tagsJSON), &n.Tags); err != nil {
			n.Tags = []string{}
		}
		if n.Tags == nil {
			n.Tags = []string{}
		}
		notes = append(notes, &n)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	return notes, nil
}

// TagInfo represents a tag with its count
type TagInfo struct {
	Name  string `json:"name"`
	Count int    `json:"count"`
}

// GetTags returns all unique tags with their note counts
func (idx *Indexer) GetTags() ([]TagInfo, error) {
	rows, err := idx.db.Query(`
		SELECT DISTINCT tag, COUNT(*) as cnt FROM (
			SELECT JSON_UNQUOTE(JSON_EXTRACT(tags, CONCAT('$[', n.n, ']'))) as tag
			FROM notes
			JOIN (SELECT 0 n UNION SELECT 1 UNION SELECT 2 UNION SELECT 3 UNION SELECT 4
			      UNION SELECT 5 UNION SELECT 6 UNION SELECT 7 UNION SELECT 8 UNION SELECT 9) n
			ON n.n < JSON_LENGTH(tags)
			WHERE tags IS NOT NULL AND tags != '[]'
		) t
		WHERE tag IS NOT NULL AND tag != ''
		GROUP BY tag
		ORDER BY cnt DESC
	`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var tags []TagInfo
	for rows.Next() {
		var t TagInfo
		if err := rows.Scan(&t.Name, &t.Count); err != nil {
			return nil, err
		}
		tags = append(tags, t)
	}
	if tags == nil {
		tags = make([]TagInfo, 0)
	}
	return tags, rows.Err()
}

// Stats represents knowledge base statistics
type Stats struct {
	TotalNotes int            `json:"total_notes"`
	ByType     map[string]int `json:"by_type"`
	ByStatus   map[string]int `json:"by_status"`
	TagCount   int            `json:"tag_count"`
	LinkCount  int            `json:"link_count"`
	RecentNotes []SearchResult `json:"recent_notes"`
}

// GetStats returns knowledge base statistics
func (idx *Indexer) GetStats() (*Stats, error) {
	stats := &Stats{
		ByType:   make(map[string]int),
		ByStatus: make(map[string]int),
	}

	// Total notes
	if err := idx.db.QueryRow("SELECT COUNT(*) FROM notes").Scan(&stats.TotalNotes); err != nil {
		return nil, fmt.Errorf("count notes: %w", err)
	}

	// By type
	rows, err := idx.db.Query("SELECT type, COUNT(*) FROM notes GROUP BY type")
	if err == nil {
		defer rows.Close()
		for rows.Next() {
			var t string
			var c int
			if err := rows.Scan(&t, &c); err != nil {
				log.Error().Err(err).Msg("stats: scan type")
				continue
			}
			stats.ByType[t] = c
		}
	}

	// By status
	rows2, err := idx.db.Query("SELECT status, COUNT(*) FROM notes GROUP BY status")
	if err == nil {
		defer rows2.Close()
		for rows2.Next() {
			var s string
			var c int
			if err := rows2.Scan(&s, &c); err != nil {
				log.Error().Err(err).Msg("stats: scan status")
				continue
			}
			stats.ByStatus[s] = c
		}
	}

	// Tag count
	_ = idx.db.QueryRow(`
		SELECT COUNT(DISTINCT tag) FROM (
			SELECT JSON_UNQUOTE(JSON_EXTRACT(tags, CONCAT('$[', n.n, ']'))) as tag
			FROM notes
			JOIN (SELECT 0 n UNION SELECT 1 UNION SELECT 2 UNION SELECT 3 UNION SELECT 4
			      UNION SELECT 5 UNION SELECT 6 UNION SELECT 7 UNION SELECT 8 UNION SELECT 9) n
			ON n.n < JSON_LENGTH(tags)
			WHERE tags IS NOT NULL AND tags != '[]'
		) t WHERE tag IS NOT NULL AND tag != ''
	`).Scan(&stats.TagCount)

	// Link count
	var linkCount int
	if err := idx.db.QueryRow("SELECT COUNT(*) FROM links").Scan(&linkCount); err != nil {
		log.Error().Err(err).Msg("stats: count links")
	}
	stats.LinkCount = linkCount

	// Recent notes
	recentRows, err := idx.db.Query(`
		SELECT id, path, title, type, status, tags, COALESCE(summary, '')
		FROM notes ORDER BY updated DESC LIMIT 5
	`)
	if err == nil {
		defer recentRows.Close()
		for recentRows.Next() {
			var r SearchResult
			var tagsJSON string
			if err := recentRows.Scan(&r.ID, &r.Path, &r.Title, &r.Type, &r.Status, &tagsJSON, &r.Summary); err != nil {
				log.Error().Err(err).Msg("stats: scan recent")
				continue
			}
			json.Unmarshal([]byte(tagsJSON), &r.Tags)
			if r.Tags == nil {
				r.Tags = []string{}
			}
			stats.RecentNotes = append(stats.RecentNotes, r)
		}
	}
	if stats.RecentNotes == nil {
		stats.RecentNotes = make([]SearchResult, 0)
	}

	return stats, nil
}
