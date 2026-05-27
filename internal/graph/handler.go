package graph

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/rs/zerolog/log"
)

type Handler struct {
	builder *Builder
}

func NewHandler(builder *Builder) *Handler {
	return &Handler{builder: builder}
}

func (h *Handler) RegisterRoutes(r *gin.RouterGroup) {
	r.GET("/graph", h.GetGraph)
	r.GET("/graph/orphans", h.GetOrphans)
}

func (h *Handler) GetGraph(c *gin.Context) {
	data, err := h.builder.Build()
	if err != nil {
		log.Error().Err(err).Msg("failed to build graph")
		c.JSON(http.StatusInternalServerError, gin.H{"error": "internal server error"})
		return
	}
	c.JSON(http.StatusOK, data)
}

func (h *Handler) GetOrphans(c *gin.Context) {
	nodes, err := h.builder.FindOrphans()
	if err != nil {
		log.Error().Err(err).Msg("failed to find orphans")
		c.JSON(http.StatusInternalServerError, gin.H{"error": "internal server error"})
		return
	}
	c.JSON(http.StatusOK, gin.H{"orphans": nodes})
}
