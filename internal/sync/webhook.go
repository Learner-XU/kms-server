package sync

import (
	"context"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"io"
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/rs/zerolog/log"

	"kms-server/internal/gitea"
	"kms-server/internal/note"
	"kms-server/internal/search"
)

type WebhookHandler struct {
	secret  string
	gitea   *gitea.Client
	noteSvc *note.Service
	indexer *search.Indexer
	sem     chan struct{}
}

func NewWebhookHandler(secret string, giteaClient *gitea.Client, noteSvc *note.Service, indexer *search.Indexer) *WebhookHandler {
	if secret == "" {
		log.Fatal().Msg("WEBHOOK_SECRET is not set — refusing to start without webhook signature verification")
	}
	return &WebhookHandler{
		secret:  secret,
		gitea:   giteaClient,
		noteSvc: noteSvc,
		indexer: indexer,
		sem:     make(chan struct{}, 3),
	}
}

func (h *WebhookHandler) RegisterRoutes(r *gin.RouterGroup) {
	r.POST("/webhooks/gitea", h.Handle)
}

func (h *WebhookHandler) Handle(c *gin.Context) {
	signature := c.GetHeader("X-Gitea-Signature")
	body, err := io.ReadAll(c.Request.Body)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "read body failed"})
		return
	}

	if !h.verifySignature(body, signature) {
		c.JSON(http.StatusForbidden, gin.H{"error": "invalid signature"})
		return
	}

	event := c.GetHeader("X-Gitea-Event")
	var payload map[string]interface{}
	if err := json.Unmarshal(body, &payload); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid json"})
		return
	}

	switch event {
	case "push":
		h.handlePush(c, payload)
	case "issues":
		c.JSON(http.StatusOK, gin.H{"status": "ok", "event": "issues"})
	case "issue_comment":
		c.JSON(http.StatusOK, gin.H{"status": "ok", "event": "issue_comment"})
	case "release":
		c.JSON(http.StatusOK, gin.H{"status": "ok", "event": "release"})
	default:
		c.JSON(http.StatusOK, gin.H{"status": "ignored", "event": event})
	}
}

func (h *WebhookHandler) handlePush(c *gin.Context, payload map[string]interface{}) {
	commits, _ := payload["commits"].([]interface{})
	var mdFiles []string

	for _, commit := range commits {
		cm, ok := commit.(map[string]interface{})
		if !ok {
			continue
		}
		for _, key := range []string{"added", "modified"} {
			files, _ := cm[key].([]interface{})
			for _, f := range files {
				path, _ := f.(string)
				if strings.HasSuffix(path, ".md") {
					mdFiles = append(mdFiles, path)
				}
			}
		}
	}

	mdFiles = unique(mdFiles)
	if len(mdFiles) > 0 {
		select {
		case h.sem <- struct{}{}:
			go func() {
				defer func() { <-h.sem }()
				h.reindexFiles(mdFiles)
			}()
		default:
			log.Warn().Int("files", len(mdFiles)).Msg("reindex skipped: too many concurrent reindex operations")
		}
	}

	c.JSON(http.StatusOK, gin.H{"status": "ok", "reindexed": len(mdFiles)})
}

func (h *WebhookHandler) reindexFiles(files []string) {
	ctx := context.Background()
	for _, f := range files {
		path := strings.TrimSuffix(f, ".md")
		n, err := h.noteSvc.Get(ctx, path)
		if err != nil {
			log.Warn().Str("path", f).Err(err).Msg("skip reindex")
			continue
		}

		indexed := &search.IndexedNote{
			ID:      n.ID,
			Path:    n.Path,
			Title:   n.Title,
			Content: n.Content,
			Type:    string(n.Type),
			Status:  string(n.Status),
			Tags:    n.Tags,
			Summary: n.Summary,
			Source:  n.Source,
			SHA:     n.SHA,
			Created: n.Created,
			Updated: n.Updated,
		}

		if err := h.indexer.UpsertNote(indexed); err != nil {
			log.Error().Str("path", f).Err(err).Msg("reindex failed")
		} else {
			log.Info().Str("path", f).Msg("reindexed")
		}
	}
}

func (h *WebhookHandler) verifySignature(payload []byte, signature string) bool {
	mac := hmac.New(sha256.New, []byte(h.secret))
	mac.Write(payload)
	expected := hex.EncodeToString(mac.Sum(nil))
	return hmac.Equal([]byte(expected), []byte(signature))
}

func unique(slice []string) []string {
	seen := make(map[string]bool)
	var result []string
	for _, s := range slice {
		if !seen[s] {
			seen[s] = true
			result = append(result, s)
		}
	}
	return result
}
