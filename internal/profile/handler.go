package profile

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/rs/zerolog/log"
)

type Handler struct {
	store *Store
}

func NewHandler(store *Store) *Handler {
	return &Handler{store: store}
}

func (h *Handler) RegisterRoutes(r *gin.RouterGroup) {
	profiles := r.Group("/profiles")
	{
		profiles.GET("", h.List)
		profiles.GET("/:username", h.Get)
		profiles.PUT("/:username", h.Update)
	}
}

// Get returns a user's profile. Public — no auth required.
func (h *Handler) Get(c *gin.Context) {
	username := c.Param("username")
	if username == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "username required"})
		return
	}

	p, err := h.store.Get(username)
	if err != nil {
		log.Error().Err(err).Str("username", username).Msg("failed to get profile")
		c.JSON(http.StatusInternalServerError, gin.H{"error": "internal server error"})
		return
	}
	if p == nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "profile not found"})
		return
	}

	c.JSON(http.StatusOK, p)
}

// List returns all profiles. Public.
func (h *Handler) List(c *gin.Context) {
	profiles, err := h.store.List()
	if err != nil {
		log.Error().Err(err).Msg("failed to list profiles")
		c.JSON(http.StatusInternalServerError, gin.H{"error": "internal server error"})
		return
	}
	c.JSON(http.StatusOK, gin.H{"profiles": profiles})
}

// Update creates or updates a profile. Auth required — must match own username.
func (h *Handler) Update(c *gin.Context) {
	targetUser := c.Param("username")
	if targetUser == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "username required"})
		return
	}

	// Must be authenticated
	currentUser, exists := c.Get("username")
	if !exists || currentUser == nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "login required"})
		return
	}
	// Must be updating own profile
	curUser, ok := currentUser.(string)
	if !ok {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "invalid user context"})
		return
	}
	if curUser != targetUser {
		c.JSON(http.StatusForbidden, gin.H{"error": "can only edit your own profile"})
		return
	}

	var p Profile
	if err := c.ShouldBindJSON(&p); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	p.Username = targetUser
	if err := h.store.Upsert(targetUser, &p); err != nil {
		log.Error().Err(err).Str("username", targetUser).Msg("failed to save profile")
		c.JSON(http.StatusInternalServerError, gin.H{"error": "internal server error"})
		return
	}

	c.JSON(http.StatusOK, p)
}
