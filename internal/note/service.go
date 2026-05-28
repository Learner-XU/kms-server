package note

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/rs/zerolog/log"

	"kms-server/internal/gitea"
	"kms-server/internal/search"
	"kms-server/pkg/frontmatter"
	"kms-server/pkg/id"
	"kms-server/pkg/markdown"
)

// sanitizePath validates a note path for traversal attacks.
// It does NOT add a prefix - the caller is responsible for ensuring
// the path is used correctly with the Gitea API.
func sanitizePath(path string) (string, error) {
	if path == "" {
		return "", fmt.Errorf("path is required")
	}
	if strings.Contains(path, "\x00") {
		return "", fmt.Errorf("invalid path: contains null byte")
	}
	if strings.HasPrefix(path, "/") {
		return "", fmt.Errorf("invalid path: must not start with /")
	}
	parts := strings.Split(path, "/")
	for _, part := range parts {
		if part == ".." {
			return "", fmt.Errorf("invalid path: contains .. traversal")
		}
	}
	return path, nil
}

type Service struct {
	gitea   *gitea.Client
	indexer *search.Indexer
}

func NewService(giteaClient *gitea.Client, indexer *search.Indexer) *Service {
	return &Service{gitea: giteaClient, indexer: indexer}
}

func (s *Service) Get(ctx context.Context, path string) (*Note, error) {
	file, err := s.gitea.GetFile(ctx, path+".md")
	if err != nil {
		return nil, err
	}
	fm, body, err := frontmatter.Parse(file.Content)
	if err != nil {
		return nil, err
	}
	if fm == nil {
		return nil, fmt.Errorf("no frontmatter in %s", path)
	}
	tags := fm.Tags
	if tags == nil {
		tags = []string{}
	}
	links := fm.Links
	if links == nil {
		links = []string{}
	}

	// Extract inline links
	inlineLinks := markdown.ExtractLinks(body)

	noteID := fm.ID
	needsWriteBack := false
	if noteID == "" {
		noteID = id.New()
		fm.ID = noteID
		needsWriteBack = true
		log.Info().Str("path", path).Str("id", noteID).Msg("auto-generated note ID")
	}

	note := &Note{
		ID:      noteID,
		Title:   fm.Title,
		Path:    path,
		Content: body,
		Tags:    tags,
		Type:    NoteType(fm.Type),
		Status:  NoteStatus(fm.Status),
		Source:  fm.Source,
		Links:   links,
		Backlinks: []string{},
		Summary: fm.Summary,
		SHA:     file.SHA,
	}

	// Add inline link targets
	for _, l := range inlineLinks {
		note.Links = appendUnique(note.Links, l.Target)
	}

	// Flexible date parsing: try RFC3339, then date-only, then datetime
	parsed := false
	for _, layout := range []string{time.RFC3339, "2006-01-02", "2006-01-02 15:04:05"} {
		if t, err := time.Parse(layout, fm.Created); err == nil {
			note.Created = t
			parsed = true
			break
		}
	}
	if !parsed {
		note.Created = time.Now()
	}
	parsed = false
	for _, layout := range []string{time.RFC3339, "2006-01-02", "2006-01-02 15:04:05"} {
		if t, err := time.Parse(layout, fm.Updated); err == nil {
			note.Updated = t
			parsed = true
			break
		}
	}
	if !parsed {
		note.Updated = note.Created
	}

	// Write back auto-generated ID (synchronous to avoid SHA conflicts)
	if needsWriteBack {
		newContent, err := frontmatter.Marshal(fm, body)
		if err != nil {
			log.Error().Err(err).Str("path", path).Msg("failed to marshal auto-generated ID")
		} else if err := s.gitea.PutFile(context.Background(), path+".md", newContent, "auto: assign note ID", file.SHA); err != nil {
			log.Error().Err(err).Str("path", path).Msg("failed to commit auto-generated ID")
		}
	}

	return note, nil
}

func (s *Service) Create(ctx context.Context, req CreateNoteRequest) (*Note, error) {
	if req.Type != "" && !isValidNoteType(req.Type) {
		return nil, fmt.Errorf("invalid note type: %s", req.Type)
	}
	noteID := id.New()
	fm := frontmatter.DefaultFrontmatter(noteID, req.Title)

	if req.Type != "" {
		fm.Type = req.Type
	}
	if req.Status != "" {
		fm.Status = req.Status
	}
	if req.Tags != nil {
		fm.Tags = req.Tags
	}

	filePath := req.Path + ".md"
	body := fmt.Sprintf("# %s\n", req.Title)
	if req.Content != "" {
		body = req.Content
	}

	content, err := frontmatter.Marshal(fm, body)
	if err != nil {
		return nil, err
	}

	if err := s.gitea.PutFile(ctx, filePath, content, "create: "+req.Title, ""); err != nil {
		return nil, err
	}

	return s.Get(ctx, req.Path)
}

func (s *Service) Update(ctx context.Context, path string, req UpdateNoteRequest) (*Note, error) {
	if req.Type != "" && !isValidNoteType(req.Type) {
		return nil, fmt.Errorf("invalid note type: %s", req.Type)
	}
	filePath := path + ".md"
	file, err := s.gitea.GetFile(ctx, filePath)
	if err != nil {
		return nil, err
	}

	fm, body, err := frontmatter.Parse(file.Content)
	if err != nil {
		return nil, err
	}

	if req.Title != "" {
		fm.Title = req.Title
	}
	if req.Content != "" {
		body = req.Content
	}
	if req.Tags != nil {
		fm.Tags = req.Tags
	}
	if req.Type != "" {
		fm.Type = req.Type
	}
	if req.Status != "" {
		fm.Status = req.Status
	}
	if req.Source != "" {
		fm.Source = req.Source
	}
	if req.Summary != "" {
		fm.Summary = req.Summary
	}
	fm.Updated = time.Now().Format(time.RFC3339)

	content, err := frontmatter.Marshal(fm, body)
	if err != nil {
		return nil, err
	}

	if err := s.gitea.PutFile(ctx, filePath, content, "update: "+fm.Title, file.SHA); err != nil {
		return nil, err
	}

	return s.Get(ctx, path)
}

func (s *Service) Delete(ctx context.Context, path string) error {
	filePath := path + ".md"
	file, err := s.gitea.GetFile(ctx, filePath)
	if err != nil {
		// If file doesn't exist in Gitea, clean up index and return nil
		if strings.Contains(err.Error(), "404") {
			if s.indexer != nil {
				_ = s.indexer.DeleteNote(path)
			}
			return nil
		}
		return err
	}
	if err := s.gitea.DeleteFile(ctx, filePath, file.SHA, "delete: "+path); err != nil {
		return err
	}
	// Clean up index
	if s.indexer != nil {
		_ = s.indexer.DeleteNote(path)
	}
	return nil
}

func (s *Service) List(ctx context.Context, dirPath string) ([]*Note, error) {
	// Try MySQL index first to avoid N+1 Gitea API calls
	if s.indexer != nil {
		notes, err := s.listFromIndex(dirPath)
		if err == nil {
			return notes, nil
		}
		if err != nil {
			log.Warn().Err(err).Msg("failed to list from index, falling back to gitea")
		}
	}
	entries, err := s.gitea.ListTree(ctx, dirPath, true)
	if err != nil {
		return nil, err
	}

	notes := make([]*Note, 0)
	for _, e := range entries {
		if e.Type != "blob" || !strings.HasSuffix(e.Path, ".md") {
			continue
		}
		notePath := strings.TrimSuffix(e.Path, ".md")
		n, err := s.Get(ctx, notePath)
		if err != nil {
			continue
		}
		notes = append(notes, n)
	}
	return notes, nil
}

func (s *Service) GetHistory(ctx context.Context, path string) ([]gitea.CommitInfo, error) {
	return s.gitea.GetFileHistory(ctx, path+".md", 1, 50)
}

func (s *Service) listFromIndex(dirPath string) ([]*Note, error) {
	indexed, err := s.indexer.ListByDir(dirPath)
	if err != nil {
		return nil, err
	}
	notes := make([]*Note, 0, len(indexed))
	for _, n := range indexed {
		notes = append(notes, &Note{
			ID: n.ID, Path: n.Path, Title: n.Title, Content: n.Content,
			Type: NoteType(n.Type), Status: NoteStatus(n.Status),
			Tags: n.Tags, Summary: n.Summary, Source: n.Source, SHA: n.SHA,
			Created: n.Created, Updated: n.Updated,
			Backlinks: []string{},
		})
	}
	return notes, nil
}

// IndexNote indexes a note in MySQL. Called after Create/Update.
func (s *Service) IndexNote(n *Note) {
	if s.indexer == nil || n == nil {
		return
	}
	idx := search.NewIndexedNote(n.ID, n.Path, n.Title, n.Content,
		string(n.Type), string(n.Status), n.Tags, n.Summary, n.Source, n.SHA, n.Created, n.Updated)
	if err := s.indexer.UpsertNote(idx); err != nil {
		log.Error().Err(err).Str("note_id", n.ID).Msg("failed to index note")
	}
}

func appendUnique(slice []string, item string) []string {
	for _, s := range slice {
		if s == item {
			return slice
		}
	}
	return append(slice, item)
}
