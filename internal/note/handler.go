package note

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/rs/zerolog/log"

	"kms-server/internal/search"
)

type Handler struct {
	svc     *Service
	indexer *search.Indexer
}

func NewHandler(svc *Service, indexer *search.Indexer) *Handler {
	return &Handler{svc: svc, indexer: indexer}
}

func (h *Handler) RegisterRoutes(r *gin.RouterGroup) {
	notes := r.Group("/notes")
	{
		notes.GET("", h.List)
		notes.POST("", h.Create)
		notes.GET("/*path", h.Get)
		notes.PUT("/*path", h.Update)
		notes.DELETE("/*path", h.Delete)
	}
	r.GET("/history/*path", h.History)
}

func (h *Handler) List(c *gin.Context) {
	dir := c.Query("dir")
	if dir != "" {
		if err := validatePathSegment(dir); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "invalid dir parameter"})
			return
		}
	}
	notes, err := h.svc.List(c.Request.Context(), dir)
	if err != nil {
		log.Error().Err(err).Msg("failed to list notes")
		c.JSON(http.StatusInternalServerError, gin.H{"error": "internal server error"})
		return
	}
	c.JSON(http.StatusOK, gin.H{"notes": notes})
}

func (h *Handler) Create(c *gin.Context) {
	var req CreateNoteRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	if err := validatePathSegment(req.Path); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	if req.Type != "" && !isValidNoteType(req.Type) {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid note type, allowed: note, daily, source, project"})
		return
	}
	n, err := h.svc.Create(c.Request.Context(), req)
	if err != nil {
		log.Error().Err(err).Msg("failed to create note")
		c.JSON(http.StatusInternalServerError, gin.H{"error": "internal server error"})
		return
	}
	// Auto-index
	h.indexNote(n)
	c.JSON(http.StatusCreated, n)
}

func (h *Handler) Get(c *gin.Context) {
	path := c.Param("path")
	if path == "" || path == "/" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "path required"})
		return
	}
	path = path[1:] // remove leading /
	safePath, err := sanitizePath(path)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	note, err := h.svc.Get(c.Request.Context(), safePath)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "note not found"})
		return
	}
	c.JSON(http.StatusOK, note)
}

func (h *Handler) Update(c *gin.Context) {
	path := c.Param("path")
	if path == "" || path == "/" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "path required"})
		return
	}
	path = path[1:]
	safePath, err := sanitizePath(path)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	var req UpdateNoteRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	if req.Type != "" && !isValidNoteType(req.Type) {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid note type, allowed: note, daily, source, project"})
		return
	}
	n, err := h.svc.Update(c.Request.Context(), safePath, req)
	if err != nil {
		log.Error().Err(err).Str("path", safePath).Msg("failed to update note")
		c.JSON(http.StatusInternalServerError, gin.H{"error": "internal server error"})
		return
	}
	// Auto-index
	h.indexNote(n)
	c.JSON(http.StatusOK, n)
}

func (h *Handler) Delete(c *gin.Context) {
	path := c.Param("path")
	if path == "" || path == "/" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "path required"})
		return
	}
	path = path[1:]
	safePath, err := sanitizePath(path)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	if err := h.svc.Delete(c.Request.Context(), safePath); err != nil {
		log.Error().Err(err).Str("path", safePath).Msg("failed to delete note")
		c.JSON(http.StatusInternalServerError, gin.H{"error": "internal server error"})
		return
	}
	c.JSON(http.StatusOK, gin.H{"status": "deleted"})
}

func (h *Handler) History(c *gin.Context) {
	path := c.Param("path")
	if path == "" || path == "/" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "path required"})
		return
	}
	path = path[1:] // remove leading /
	safePath, err := sanitizePath(path)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	commits, err := h.svc.GetHistory(c.Request.Context(), safePath)
	if err != nil {
		log.Error().Err(err).Str("path", safePath).Msg("failed to get history")
		c.JSON(http.StatusInternalServerError, gin.H{"error": "internal server error"})
		return
	}
	c.JSON(http.StatusOK, gin.H{"commits": commits})
}

func isValidNoteType(t string) bool {
	switch NoteType(t) {
	case TypeNote, TypeDaily, TypeSource, TypeProject:
		return true
	}
	return false
}

func (h *Handler) indexNote(n *Note) {
	if h.indexer == nil || n == nil {
		return
	}
	idx := search.NewIndexedNote(n.ID, n.Path, n.Title, n.Content,
		string(n.Type), string(n.Status), n.Tags, n.Summary, n.Source, n.SHA, n.Created, n.Updated)
	if err := h.indexer.UpsertNote(idx); err != nil {
		// Log but don't fail the request
		log.Error().Err(err).Str("note_id", n.ID).Msg("failed to index note")
		return
	}
}
