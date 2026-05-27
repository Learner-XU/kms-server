package main

import (
	"fmt"
	"database/sql"

	"github.com/gin-gonic/gin"
	"github.com/rs/zerolog/log"

	"kms-server/internal/auth"
	"kms-server/internal/config"
	"kms-server/internal/gitea"
	"kms-server/internal/graph"
	"kms-server/internal/middleware"
	"kms-server/internal/note"
	"kms-server/internal/search"
	"kms-server/internal/sync"
)

func main() {
	cfg := config.Load()

	// Gitea client
	giteaClient := gitea.NewClient(cfg.Gitea.URL, cfg.Gitea.Token, cfg.Gitea.Repo)

	// MySQL connection (shared by indexer and auth)
	db, err := sql.Open("mysql", cfg.MySQL.DSN)
	if err != nil {
		log.Fatal().Err(err).Msg("failed to open MySQL")
	}
	defer db.Close()

	// MySQL indexer
	indexer, err := search.NewIndexerWithDB(db)
	if err != nil {
		log.Fatal().Err(err).Msg("failed to init MySQL indexer")
	}

	// Services
	noteSvc := note.NewService(giteaClient)
	noteSvc.SetIndexer(indexer)
	graphBuilder := graph.NewBuilder(indexer.DB())

	// Auth
	authSvc := auth.NewService(db, cfg.JWTSecret)
	authHandler := auth.NewHandler(authSvc)

	// Handlers
	noteHandler := note.NewHandler(noteSvc, indexer)
	searchHandler := search.NewHandler(indexer)
	graphHandler := graph.NewHandler(graphBuilder)
	webhookHandler := sync.NewWebhookHandler(cfg.Webhook.Secret, giteaClient, noteSvc, indexer)

	// Startup reindex — scan all .md files and populate MySQL
	go sync.ReindexAll(giteaClient, noteSvc, indexer)

	// Router
	r := gin.Default()
	r.Use(middleware.CORS())
	r.Use(middleware.JWTAuth(authSvc.JWTManager()))

	api := r.Group("/api")
	{
		authHandler.RegisterRoutes(api)
		noteHandler.RegisterRoutes(api)
		searchHandler.RegisterRoutes(api)
		graphHandler.RegisterRoutes(api)
	}

	webhookHandler.RegisterRoutes(r.Group("/"))

	// Health check
	r.GET("/health", func(c *gin.Context) {
		c.JSON(200, gin.H{"status": "ok"})
	})

	addr := fmt.Sprintf(":%s", cfg.Server.Port)
	log.Info().Str("addr", addr).Msg("KMS server starting")
	if err := r.Run(addr); err != nil {
		log.Fatal().Err(err).Msg("server failed")
	}
}
