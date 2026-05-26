package note

import (
	"time"
)

type NoteType string

const (
	TypeNote    NoteType = "note"
	TypeDaily   NoteType = "daily"
	TypeSource  NoteType = "source"
	TypeProject NoteType = "project"
)

type NoteStatus string

const (
	StatusSeed     NoteStatus = "seed"
	StatusGrowing  NoteStatus = "growing"
	StatusMature   NoteStatus = "mature"
	StatusArchived NoteStatus = "archived"
)

type Note struct {
	ID        string     `json:"id"`
	Title     string     `json:"title"`
	Path      string     `json:"path"`
	Content   string     `json:"content"`
	Tags      []string   `json:"tags"`
	Type      NoteType   `json:"type"`
	Status    NoteStatus `json:"status"`
	Source    string     `json:"source,omitempty"`
	Links     []string   `json:"links"`
	Backlinks []string   `json:"backlinks"`
	Summary   string     `json:"summary,omitempty"`
	Created   time.Time  `json:"created"`
	Updated   time.Time  `json:"updated"`
	SHA       string     `json:"sha"`
}

type Frontmatter struct {
	ID      string   `yaml:"id"`
	Title   string   `yaml:"title"`
	Created string   `yaml:"created"`
	Updated string   `yaml:"updated"`
	Tags    []string `yaml:"tags"`
	Type    string   `yaml:"type"`
	Status  string   `yaml:"status"`
	Source  string   `yaml:"source,omitempty"`
	Links   []string `yaml:"links,omitempty"`
	Summary string   `yaml:"summary,omitempty"`
}

type Link struct {
	SourceID    string `json:"source_id"`
	TargetID    string `json:"target_id"`
	SourcePath  string `json:"source_path"`
	TargetTitle string `json:"target_title"`
	Context     string `json:"context,omitempty"`
}

type GraphNode struct {
	ID        string   `json:"id"`
	Title     string   `json:"title"`
	Type      NoteType `json:"type"`
	Status    string   `json:"status"`
	Tags      []string `json:"tags"`
	LinkCount int      `json:"link_count"`
	Updated   string   `json:"updated"`
}

type GraphEdge struct {
	Source  string  `json:"source"`
	Target  string  `json:"target"`
	Weight  float64 `json:"weight"`
	Context string  `json:"context,omitempty"`
}

type GraphData struct {
	Nodes []GraphNode `json:"nodes"`
	Edges []GraphEdge `json:"edges"`
}

type CreateNoteRequest struct {
	Title   string   `json:"title" binding:"required"`
	Path    string   `json:"path" binding:"required"`
	Content string   `json:"content"`
	Tags    []string `json:"tags"`
	Type    string   `json:"type"`
	Status  string   `json:"status"`
}

type UpdateNoteRequest struct {
	Title   string   `json:"title"`
	Content string   `json:"content"`
	Tags    []string `json:"tags"`
	Type    string   `json:"type"`
	Status  string   `json:"status"`
	Source  string   `json:"source"`
	Summary string   `json:"summary"`
}
