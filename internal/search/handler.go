package search

import (
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"
)

type Handler struct {
	indexer *Indexer
}

func NewHandler(indexer *Indexer) *Handler {
	return &Handler{indexer: indexer}
}

func (h *Handler) RegisterRoutes(r *gin.RouterGroup) {
	r.GET("/search", h.Search)
	r.GET("/notes/:id/backlinks", h.Backlinks)
}

func (h *Handler) Search(c *gin.Context) {
	q := c.Query("q")
	if q == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "q parameter required"})
		return
	}

	filters := SearchFilters{
		Type:   c.Query("type"),
		Status: c.Query("status"),
	}
	if tags := c.Query("tags"); tags != "" {
		filters.Tags = strings.Split(tags, ",")
	}

	limit := 20
	offset := 0

	results, total, err := h.indexer.Search(q, filters, limit, offset)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"results": results,
		"total":   total,
		"query":   q,
	})
}

func (h *Handler) Backlinks(c *gin.Context) {
	noteID := c.Param("id")
	results, err := h.indexer.GetBacklinks(noteID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"backlinks": results})
}
