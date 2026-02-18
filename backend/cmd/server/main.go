package main

import (
	"context"
	"log"
	"net/http"
	"os/signal"
	"syscall"
	"time"

	"github.com/cds-id/pdt/backend/internal/config"
	"github.com/cds-id/pdt/backend/internal/crypto"
	"github.com/cds-id/pdt/backend/internal/database"
	"github.com/cds-id/pdt/backend/internal/handlers"
	"github.com/cds-id/pdt/backend/internal/middleware"
	"github.com/cds-id/pdt/backend/internal/services/report"
	"github.com/cds-id/pdt/backend/internal/services/storage"
	"github.com/cds-id/pdt/backend/internal/worker"
	"github.com/gin-gonic/gin"
	"github.com/gin-contrib/cors"
)

func main() {
	cfg, err := config.Load()
	if err != nil {
		log.Fatalf("failed to load config: %v", err)
	}

	db, err := database.Connect(cfg.DSN())
	if err != nil {
		log.Fatalf("failed to connect to database: %v", err)
	}

	if err := database.Migrate(db); err != nil {
		log.Fatalf("failed to migrate database: %v", err)
	}

	encryptor, err := crypto.NewEncryptor(cfg.EncryptionKey)
	if err != nil {
		log.Fatalf("failed to create encryptor: %v", err)
	}

	// Graceful shutdown context (needed by both worker and server)
	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer stop()

	// R2 storage (optional)
	var r2Client *storage.R2Client
	if cfg.R2AccountID != "" && cfg.R2AccessKeyID != "" {
		r2Client = storage.NewR2Client(cfg.R2AccountID, cfg.R2AccessKeyID, cfg.R2SecretAccessKey, cfg.R2BucketName, cfg.R2PublicDomain)
		log.Printf("R2 storage configured: bucket=%s, domain=%s", cfg.R2BucketName, cfg.R2PublicDomain)
	}

	// Worker scheduler
	var syncStatus *worker.SyncStatus
	if cfg.SyncEnabled {
		scheduler := worker.NewScheduler(db, encryptor, cfg.SyncIntervalCommits, cfg.SyncIntervalJira, cfg.ReportAutoGenerate, cfg.ReportAutoTime, r2Client)
		scheduler.Start(ctx)
		syncStatus = scheduler.Status
	} else {
		syncStatus = worker.NewSyncStatus()
	}

	// Handlers
	authHandler := &handlers.AuthHandler{
		DB:             db,
		JWTSecret:      cfg.JWTSecret,
		JWTExpiryHours: cfg.JWTExpiryHours,
	}
	userHandler := &handlers.UserHandler{DB: db, Encryptor: encryptor}
	repoHandler := &handlers.RepoHandler{DB: db}
	syncHandler := &handlers.SyncHandler{DB: db, Encryptor: encryptor, Status: syncStatus}
	commitHandler := &handlers.CommitHandler{DB: db}
	jiraHandler := &handlers.JiraHandler{DB: db, Encryptor: encryptor}
	reportGen := report.NewGenerator(db, encryptor)
	reportHandler := &handlers.ReportHandler{DB: db, Generator: reportGen, R2: r2Client}

	// Router
	r := gin.Default()

	// CORS middleware
	r.Use(cors.New(cors.Config{
		AllowOrigins:     []string{"*"},
		AllowMethods:     []string{"GET", "POST", "PUT", "PATCH", "DELETE", "OPTIONS"},
		AllowHeaders:     []string{"Origin", "Content-Type", "Accept", "Authorization"},
		ExposeHeaders:    []string{"Content-Length"},
		AllowCredentials: true,
		MaxAge:           12 * time.Hour,
	}))

	api := r.Group("/api")
	{
		auth := api.Group("/auth")
		{
			auth.POST("/register", authHandler.Register)
			auth.POST("/login", authHandler.Login)
		}

		protected := api.Group("")
		protected.Use(middleware.JWTAuth(cfg.JWTSecret))
		{
			user := protected.Group("/user")
			{
				user.GET("/profile", userHandler.GetProfile)
				user.PUT("/profile", userHandler.UpdateProfile)
				user.POST("/profile/validate", userHandler.ValidateConnections)
			}

			repos := protected.Group("/repos")
			{
				repos.GET("", repoHandler.List)
				repos.POST("", repoHandler.Add)
				repos.DELETE("/:id", repoHandler.Delete)
				repos.POST("/:id/validate", repoHandler.Validate)
			}

			protected.POST("/sync/commits", syncHandler.SyncCommits)
			protected.GET("/sync/status", syncHandler.SyncStatus)

			commits := protected.Group("/commits")
			{
				commits.GET("", commitHandler.List)
				commits.GET("/missing", commitHandler.Missing)
				commits.POST("/:sha/link", commitHandler.Link)
			}

			jira := protected.Group("/jira")
			{
				jira.GET("/sprints", jiraHandler.ListSprints)
				jira.GET("/sprints/:id", jiraHandler.GetSprint)
				jira.GET("/active-sprint", jiraHandler.GetActiveSprint)
				jira.GET("/cards", jiraHandler.ListCards)
				jira.GET("/cards/:key", jiraHandler.GetCard)
			}

			reports := protected.Group("/reports")
			{
				reports.POST("/generate", reportHandler.Generate)
				reports.GET("", reportHandler.List)
				reports.GET("/:id", reportHandler.Get)
				reports.DELETE("/:id", reportHandler.Delete)

				templates := reports.Group("/templates")
				{
					templates.GET("", reportHandler.ListTemplates)
					templates.POST("", reportHandler.CreateTemplate)
					templates.PUT("/:id", reportHandler.UpdateTemplate)
					templates.DELETE("/:id", reportHandler.DeleteTemplate)
					templates.POST("/preview", reportHandler.PreviewTemplate)
				}
			}
		}
	}

	// Graceful shutdown
	srv := &http.Server{
		Addr:    ":" + cfg.ServerPort,
		Handler: r,
	}

	go func() {
		log.Printf("server starting on :%s", cfg.ServerPort)
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatalf("listen failed: %v", err)
		}
	}()

	<-ctx.Done()
	log.Println("shutting down...")

	shutdownCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if err := srv.Shutdown(shutdownCtx); err != nil {
		log.Fatalf("shutdown failed: %v", err)
	}

	sqlDB, _ := db.DB()
	sqlDB.Close()

	log.Println("server stopped")
}
