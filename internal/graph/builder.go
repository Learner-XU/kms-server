package graph

import (
	"database/sql"
	"encoding/json"

	"kms-server/internal/note"
)

type Builder struct {
	db *sql.DB
}

func NewBuilder(db *sql.DB) *Builder {
	return &Builder{db: db}
}

func (b *Builder) Build() (*note.GraphData, error) {
	nodes, err := b.buildNodes()
	if err != nil {
		return nil, err
	}
	edges, err := b.buildEdges()
	if err != nil {
		return nil, err
	}
	return &note.GraphData{Nodes: nodes, Edges: edges}, nil
}

func (b *Builder) buildNodes() ([]note.GraphNode, error) {
	rows, err := b.db.Query(`
		SELECT n.id, n.title, n.type, n.status, n.tags, n.updated,
			   (SELECT COUNT(*) FROM links WHERE source_id = n.id OR target_id = n.id) as link_count
		FROM notes n
		ORDER BY link_count DESC
	`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var nodes []note.GraphNode
	for rows.Next() {
		var n note.GraphNode
		var tagsJSON, updated string
		if err := rows.Scan(&n.ID, &n.Title, &n.Type, &n.Status, &tagsJSON, &updated, &n.LinkCount); err != nil {
			return nil, err
		}
		json.Unmarshal([]byte(tagsJSON), &n.Tags)
		n.Updated = updated
		nodes = append(nodes, n)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	return nodes, nil
}

func (b *Builder) buildEdges() ([]note.GraphEdge, error) {
	rows, err := b.db.Query(`SELECT source_id, target_id, context FROM links`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var edges []note.GraphEdge
	for rows.Next() {
		var e note.GraphEdge
		if err := rows.Scan(&e.Source, &e.Target, &e.Context); err != nil {
			return nil, err
		}
		e.Weight = 1.0
		edges = append(edges, e)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	return edges, nil
}

func (b *Builder) FindOrphans() ([]note.GraphNode, error) {
	rows, err := b.db.Query(`
		SELECT id, title, type, status, tags, updated, 0 as link_count
		FROM notes
		WHERE id NOT IN (SELECT source_id FROM links UNION SELECT target_id FROM links)
		ORDER BY updated DESC
	`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var nodes []note.GraphNode
	for rows.Next() {
		var n note.GraphNode
		var tagsJSON, updated string
		if err := rows.Scan(&n.ID, &n.Title, &n.Type, &n.Status, &tagsJSON, &updated, &n.LinkCount); err != nil {
			return nil, err
		}
		json.Unmarshal([]byte(tagsJSON), &n.Tags)
		n.Updated = updated
		nodes = append(nodes, n)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	return nodes, nil
}
