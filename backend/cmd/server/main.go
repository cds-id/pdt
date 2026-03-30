package main

import (
	"context"
	"log"
	"net/http"
	"os/signal"
	"syscall"
	"time"

	"github.com/cds-id/pdt/backend/internal/ai/minimax"
	"github.com/cds-id/pdt/backend/internal/config"
	"github.com/cds-id/pdt/backend/internal/crypto"
	"github.com/cds-id/pdt/backend/internal/database"
	"github.com/cds-id/pdt/backend/internal/handlers"
	"github.com/cds-id/pdt/backend/internal/middleware"
	"github.com/cds-id/pdt/backend/internal/services/report"
	"github.com/cds-id/pdt/backend/internal/services/storage"
	waService "github.com/cds-id/pdt/backend/internal/services/whatsapp"
	wvService "github.com/cds-id/pdt/backend/internal/services/weaviate"
	"github.com/cds-id/pdt/backend/internal/worker"
	"github.com/gin-contrib/cors"
	"github.com/gin-gonic/gin"
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

	// Weaviate client (optional)
	var weaviateClient *wvService.Client
	var embeddingWorker *wvService.EmbeddingWorker
	if cfg.GeminiAPIKey != "" {
		weaviateClient = wvService.NewClient(cfg.WeaviateURL, cfg.GeminiAPIKey)
		if weaviateClient.IsAvailable() {
			embeddingWorker = wvService.NewEmbeddingWorker(weaviateClient, db)
			embeddingWorker.Start(ctx)
			log.Printf("Weaviate connected: %s", cfg.WeaviateURL)
		}
	}

	// WhatsApp manager
	var waManager *waService.Manager
	waManager, err = waService.NewManager(ctx, db, r2Client, embeddingWorker, cfg.WhatsmeowDBPath, cfg.MistralAPIKey)
	if err != nil {
		log.Printf("WhatsApp manager init failed: %v", err)
	} else {
		waManager.Start(ctx)
		sender := waService.NewSenderWorker(db, waManager)
		sender.Start(ctx)
	}

	// Worker scheduler
	var syncStatus *worker.SyncStatus
	if cfg.SyncEnabled {
		scheduler := worker.NewScheduler(db, encryptor, cfg.SyncIntervalCommits, cfg.SyncIntervalJira, cfg.ReportAutoGenerate, cfg.ReportAutoTime, cfg.ReportMonthlyAutoTime, r2Client)
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

	var miniMaxClient *minimax.Client
	if cfg.MiniMaxAPIKey != "" {
		miniMaxClient = minimax.NewClient(cfg.MiniMaxAPIKey, cfg.MiniMaxGroupID)
	}

	aiUsageHandler := &handlers.AIUsageHandler{DB: db}

	chatHandler := &handlers.ChatHandler{
		DB:              db,
		MiniMaxClient:   miniMaxClient,
		Encryptor:       encryptor,
		R2:              r2Client,
		ReportGenerator: reportGen,
		ContextWindow:   cfg.AIContextWindow,
		WaManager:       waManager,
		WeaviateClient:  weaviateClient,
	}

	waHandler := &handlers.WhatsAppHandler{
		DB:      db,
		Manager: waManager,
	}

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
			protected.POST("/sync/jira", syncHandler.SyncJira)
			protected.GET("/sync/status", syncHandler.SyncStatus)

			commits := protected.Group("/commits")
			{
				commits.GET("", commitHandler.List)
				commits.GET("/missing", commitHandler.Missing)
				commits.POST("/:sha/link", commitHandler.Link)
			}

			jira := protected.Group("/jira")
			{
				jira.GET("/workspaces", jiraHandler.ListWorkspaces)
				jira.POST("/workspaces", jiraHandler.AddWorkspace)
				jira.PATCH("/workspaces/:id", jiraHandler.UpdateWorkspace)
				jira.DELETE("/workspaces/:id", jiraHandler.DeleteWorkspace)
				jira.GET("/sprints", jiraHandler.ListSprints)
				jira.GET("/sprints/:id", jiraHandler.GetSprint)
				jira.GET("/active-sprint", jiraHandler.GetActiveSprint)
				jira.GET("/cards", jiraHandler.ListCards)
				jira.GET("/cards/:key", jiraHandler.GetCard)
				jira.GET("/cards/:key/comments", jiraHandler.GetCardComments)
			}

			reports := protected.Group("/reports")
			{
				reports.POST("/generate", reportHandler.Generate)
				reports.POST("/generate/monthly", reportHandler.GenerateMonthly)
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

			protected.GET("/ai/usage", aiUsageHandler.GetUsageSummary)

			if miniMaxClient != nil {
				protected.GET("/ws/chat", chatHandler.HandleWebSocket)
				protected.GET("/conversations", chatHandler.ListConversations)
				protected.GET("/conversations/:id", chatHandler.GetConversation)
				protected.DELETE("/conversations/:id", chatHandler.DeleteConversation)
			}

			wa := protected.Group("/wa")
			{
				waNumbers := wa.Group("/numbers")
				{
					waNumbers.GET("", waHandler.ListNumbers)
					waNumbers.POST("", waHandler.AddNumber)
					waNumbers.PATCH("/:id", waHandler.UpdateNumber)
					waNumbers.DELETE("/:id", waHandler.DeleteNumber)
					waNumbers.POST("/:id/disconnect", waHandler.DisconnectNumber)
					waNumbers.GET("/:id/groups", waHandler.GetGroups)
					waNumbers.GET("/:id/contacts", waHandler.GetContacts)
					waNumbers.GET("/:id/listeners", waHandler.ListListeners)
					waNumbers.POST("/:id/listeners", waHandler.AddListener)
				}

				waListeners := wa.Group("/listeners")
				{
					waListeners.PATCH("/:id", waHandler.UpdateListener)
					waListeners.DELETE("/:id", waHandler.DeleteListener)
					waListeners.GET("/:id/messages", waHandler.ListMessages)
				}

				wa.GET("/messages/search", waHandler.SearchMessages)

				waOutbox := wa.Group("/outbox")
				{
					waOutbox.GET("", waHandler.ListOutbox)
					waOutbox.PATCH("/:id", waHandler.UpdateOutbox)
					waOutbox.DELETE("/:id", waHandler.DeleteOutbox)
				}

				wa.GET("/pair/:id", waHandler.HandlePairing)
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

	if waManager != nil {
		waManager.Shutdown()
	}

	if err := srv.Shutdown(shutdownCtx); err != nil {
		log.Fatalf("shutdown failed: %v", err)
	}

	sqlDB, _ := db.DB()
	sqlDB.Close()

	log.Println("server stopped")
}
