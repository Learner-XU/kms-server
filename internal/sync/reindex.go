package sync

import (
	"context"
	"strings"

	"github.com/rs/zerolog/log"

	"kms-server/internal/gitea"
	"kms-server/internal/note"
	"kms-server/internal/search"
)

// ReindexAll scans all .md files in Gitea and indexes them into MySQL.
// Uses two passes: first collect all notes, then resolve link targets.
func ReindexAll(giteaClient *gitea.Client, noteSvc *note.Service, indexer *search.Indexer) {
	ctx := context.Background()

	entries, err := giteaClient.ListTree(ctx, "", true)
	if err != nil {
		log.Error().Err(err).Msg("reindex: failed to list tree")
		return
	}

	// Pass 1: Index all notes (without link resolution)
	type noteInfo struct {
		note   *note.Note
		links  []string // link target titles
	}
	var allNotes []noteInfo

	for _, e := range entries {
		if e.Type != "blob" || !strings.HasSuffix(e.Path, ".md") {
			continue
		}
		path := strings.TrimSuffix(e.Path, ".md")
		n, err := noteSvc.Get(ctx, path)
		if err != nil {
			log.Warn().Str("path", path).Err(err).Msg("reindex: skip")
			continue
		}

		idx := search.NewIndexedNote(n.ID, n.Path, n.Title, n.Content, string(n.Type), string(n.Status), n.Tags, n.Summary, n.Source, n.SHA, n.Created, n.Updated)

		if err := indexer.UpsertNote(idx); err != nil {
			log.Error().Str("path", path).Err(err).Msg("reindex: upsert failed")
			continue
		}

		allNotes = append(allNotes, noteInfo{note: n, links: n.Links})
	}

	// Pass 2: Resolve link targets by title and build edges
	titleToID := make(map[string]string)
	for _, ni := range allNotes {
		titleToID[ni.note.Title] = ni.note.ID
	}

	for _, ni := range allNotes {
		for _, linkTitle := range ni.links {
			targetID, ok := titleToID[linkTitle]
			if !ok {
				continue // unresolved link
			}
			// Insert link edge directly
			_, err := indexer.DB().Exec(`
				INSERT IGNORE INTO links (source_id, target_id, target_title, context)
				VALUES (?, ?, ?, '')
			`, ni.note.ID, targetID, linkTitle)
			if err != nil {
				log.Error().Str("from", ni.note.Title).Str("to", linkTitle).Err(err).Msg("reindex: link insert failed")
			}
		}
	}

	log.Info().Int("notes", len(allNotes)).Msg("startup reindex complete")
}
