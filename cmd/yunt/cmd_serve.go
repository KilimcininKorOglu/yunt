package main

import (
	"context"
	"fmt"
	"os"
	"os/signal"
	"sync"
	"syscall"

	"github.com/spf13/cobra"

	"yunt/internal/api"
	"yunt/internal/api/handlers"
	"yunt/internal/api/middleware"
	"yunt/internal/config"
	imapserver "yunt/internal/imap"
	"yunt/internal/parser"
	"yunt/internal/storage"
	"yunt/internal/repository/factory"
	"yunt/internal/repository/mongodb"
	"yunt/internal/repository/mysql"
	"yunt/internal/repository/postgres"
	"yunt/internal/repository/sqlite"
	"yunt/internal/service"
	smtpserver "yunt/internal/smtp"
	"yunt/webui"
)

var (
	smtpPort    int
	imapPort    int
	apiPort     int
	enableSMTP  bool
	enableIMAP  bool
	enableAPI   bool
)

var serveCmd = &cobra.Command{
	Use:   "serve",
	Short: "Start the Yunt mail server",
	Long: `Start the Yunt mail server with SMTP, IMAP, and API services.

By default, all services are started according to the configuration file.
You can override specific port settings using command-line flags.

Examples:
  yunt serve
  yunt serve --config /path/to/yunt.yaml
  yunt serve --smtp-port 2525 --api-port 8080`,
	RunE: runServe,
}

func init() {
	serveCmd.Flags().IntVar(&smtpPort, "smtp-port", 0, "override SMTP server port")
	serveCmd.Flags().IntVar(&imapPort, "imap-port", 0, "override IMAP server port")
	serveCmd.Flags().IntVar(&apiPort, "api-port", 0, "override API server port")
	serveCmd.Flags().BoolVar(&enableSMTP, "smtp", false, "start only SMTP server")
	serveCmd.Flags().BoolVar(&enableIMAP, "imap", false, "start only IMAP server")
	serveCmd.Flags().BoolVar(&enableAPI, "api", false, "start only API/Web UI server")
}

func runServe(cmd *cobra.Command, args []string) error {
	log := getLogger()
	cfg := getConfig()

	if smtpPort > 0 {
		cfg.SMTP.Port = smtpPort
	}
	if imapPort > 0 {
		cfg.IMAP.Port = imapPort
	}
	if apiPort > 0 {
		cfg.API.Port = apiPort
	}

	// Selective service flags: if any --smtp/--imap/--api flag is set,
	// only start those services (disable the rest)
	if enableSMTP || enableIMAP || enableAPI {
		cfg.SMTP.Enabled = enableSMTP
		cfg.IMAP.Enabled = enableIMAP
		cfg.API.Enabled = enableAPI
	}

	log.Info().
		Str("version", version).
		Str("commit", commit).
		Msg("Starting Yunt mail server")

	// Initialize repository
	repoFactory, err := factory.New(&cfg.Database)
	if err != nil {
		return fmt.Errorf("failed to create repository factory: %w", err)
	}

	repo, err := repoFactory.Create()
	if err != nil {
		return fmt.Errorf("failed to create repository: %w", err)
	}
	defer repo.Close()

	log.Info().Str("driver", cfg.Database.Driver).Msg("Database connected")

	// Initialize storage backend for attachments
	storageBackend, err := storage.NewFromConfig(context.Background(), cfg.Storage)
	if err != nil {
		return fmt.Errorf("failed to create storage backend: %w", err)
	}
	if storageBackend != nil {
		log.Info().Str("type", cfg.Storage.Type).Msg("Storage backend initialized")
		switch r := repo.(type) {
		case *sqlite.Repository:
			r.Attachments().(*sqlite.AttachmentRepository).SetStorageBackend(storageBackend)
		case *postgres.Repository:
			r.Attachments().(*postgres.AttachmentRepository).SetStorageBackend(storageBackend)
		case *mysql.Repository:
			r.Attachments().(*mysql.AttachmentRepository).SetStorageBackend(storageBackend)
		case *mongodb.Repository:
			r.Attachments().(*mongodb.AttachmentRepository).SetStorageBackend(storageBackend)
		}
	}

	// Initialize session store (DB-backed for all drivers)
	var sessionStore service.SessionStore
	switch r := repo.(type) {
	case *sqlite.Repository:
		sessionStore = sqlite.NewDBSessionStore(r.DB())
	case *postgres.Repository:
		sessionStore = postgres.NewDBSessionStore(r.DB())
	case *mysql.Repository:
		sessionStore = mysql.NewDBSessionStore(r.DB())
	case *mongodb.Repository:
		sessionStore = mongodb.NewDBSessionStore(r)
	default:
		sessionStore = service.NewInMemorySessionStore()
	}

	// Initialize services
	authService := service.NewAuthService(cfg.Auth, repo.Users(), sessionStore)
	userService := service.NewUserService(cfg.Auth, repo.Users())
	mailboxService := service.NewMailboxService(repo, nil)
	messageService := service.NewMessageService(repo, nil)
	p := parser.NewParser()
	p.MaxMessageSize = cfg.SMTP.MaxMessageSize
	p.MaxAttachmentSize = cfg.SMTP.MaxAttachmentSize
	messageService.WithParser(p)
	webhookService := service.NewWebhookService(repo, nil)
	userService.WithWebhookService(webhookService)
	notifyService := service.NewNotifyService()
	messageService.WithNotifyService(notifyService)

	// Context for coordinated shutdown
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)

	errChan := make(chan error, 3)

	// Start SMTP server
	if cfg.SMTP.Enabled {
		smtpCfg, smtpErr := smtpserver.NewConfig(cfg)
		if smtpErr != nil {
			return fmt.Errorf("invalid SMTP config: %w", smtpErr)
		}

		smtpSrv, smtpErr := smtpserver.New(smtpCfg, log.Logger,
			smtpserver.WithRepo(repo),
			smtpserver.WithMailboxRepo(repo.Mailboxes()),
			smtpserver.WithMessageRepo(repo.Messages()),
			smtpserver.WithAttachmentRepo(repo.Attachments()),
			smtpserver.WithNotifyService(notifyService),
			smtpserver.WithWebhookService(webhookService),
		)
		if smtpErr != nil {
			return fmt.Errorf("failed to create SMTP server: %w", smtpErr)
		}

		go func() {
			log.Info().Str("addr", fmt.Sprintf("%s:%d", cfg.SMTP.Host, cfg.SMTP.Port)).Msg("SMTP server starting")
			if err := smtpSrv.Start(); err != nil {
				errChan <- fmt.Errorf("SMTP server error: %w", err)
			}
		}()

		defer func() {
			shutdownCtx, shutdownCancel := context.WithTimeout(context.Background(), cfg.Server.GracefulTimeout)
			defer shutdownCancel()
			_ = smtpSrv.Stop(shutdownCtx)
		}()
	}

	// Start IMAP server
	if cfg.IMAP.Enabled {
		imapCfg := imapserver.NewConfigFromApp(cfg)
		imapSrv, imapErr := imapserver.NewServer(imapCfg, log.Logger)
		if imapErr != nil {
			return fmt.Errorf("failed to create IMAP server: %w", imapErr)
		}

		imapBackendCfg := &imapserver.BackendConfig{}
		imapBackend := imapserver.NewBackend(repo, log.Logger, imapBackendCfg)
		imapSrv.SetBackend(imapBackend)

		go func() {
			log.Info().Str("addr", fmt.Sprintf("%s:%d", cfg.IMAP.Host, cfg.IMAP.Port)).Msg("IMAP server starting")
			if err := imapSrv.Start(ctx); err != nil {
				errChan <- fmt.Errorf("IMAP server error: %w", err)
			}
		}()

		defer func() {
			shutdownCtx, shutdownCancel := context.WithTimeout(context.Background(), cfg.Server.GracefulTimeout)
			defer shutdownCancel()
			_ = imapSrv.Stop(shutdownCtx)
		}()
	}

	// Start API server
	if cfg.API.Enabled {
		apiSrv := api.New(cfg.API, api.WithLogger(log))
		api.SetVersion(version)

		v1 := apiSrv.Router().V1()

		authMiddleware := middleware.Auth(authService)

		authHandler := handlers.NewAuthHandler(authService)
		authHandler.RegisterRoutes(v1)

		authed := v1.Group("", authMiddleware)

		messageHandler := handlers.NewMessageHandler(messageService, mailboxService, authService)
		messageHandler.RegisterRoutes(authed)

		mailboxHandler := handlers.NewMailboxHandler(mailboxService, authService)
		mailboxHandler.RegisterRoutes(authed)

		userHandler := handlers.NewUsersHandler(userService, authService)
		userHandler.RegisterRoutes(authed, authService)

		webhookHandler := handlers.NewWebhookHandler(webhookService, authService)
		webhookHandler.RegisterRoutes(authed)

		attachmentHandler := handlers.NewAttachmentHandler(messageService, authService)
		attachmentHandler.RegisterRoutes(authed)

		searchHandler := handlers.NewSearchHandler(messageService, authService)
		searchHandler.RegisterRoutes(authed)

		healthHandler := handlers.NewHealthHandler(repo, version)
		healthHandler.RegisterRoutes(apiSrv.Echo())
		healthHandler.RegisterAPIRoutes(v1)

		eventHandler := handlers.NewEventHandler(notifyService, authService, repo.Mailboxes())
		eventHandler.RegisterRoutes(v1)

		settingsHandler := handlers.NewSettingsHandler(repo.Settings(), authService)
		settingsHandler.RegisterRoutes(v1)

		systemHandler := handlers.NewSystemHandler(handlers.SystemHandlerConfig{
			Repo:           repo,
			AuthService:    authService,
			MessageService: messageService,
			Config:         cfg,
			Version:        version,
		})
		systemHandler.RegisterRoutes(authed)

		if webui.IsAvailable() {
			apiSrv.Echo().GET("/*", webui.Handler())
			log.Info().Msg("Web UI enabled")
		}

		go func() {
			if err := apiSrv.StartWithContext(ctx); err != nil {
				errChan <- fmt.Errorf("API server error: %w", err)
			}
		}()

		defer func() {
			shutdownCtx, shutdownCancel := context.WithTimeout(context.Background(), cfg.Server.GracefulTimeout)
			defer shutdownCancel()
			_ = apiSrv.Shutdown(shutdownCtx)
		}()
	}

	// Print startup banner
	printBanner(cfg)

	// Wait for shutdown signal or server error
	select {
	case sig := <-sigChan:
		log.Info().Str("signal", sig.String()).Msg("Received shutdown signal")
	case err := <-errChan:
		log.Error().Err(err).Msg("Server error")
		cancel()
		return err
	}

	log.Info().Dur("timeout", cfg.Server.GracefulTimeout).Msg("Initiating graceful shutdown")
	cancel()

	// Give deferred shutdown functions time to complete
	var wg sync.WaitGroup
	wg.Add(1)
	go func() {
		defer wg.Done()
	}()
	wg.Wait()

	log.Info().Msg("Server stopped successfully")
	return nil
}

func printBanner(cfg *config.Config) {
	fmt.Println()
	fmt.Println("=================================================")
	fmt.Println("  Yunt - Development Mail Server")
	fmt.Printf("  Version: %s (commit: %s)\n", version, commit)
	fmt.Println("=================================================")
	fmt.Println()

	if cfg.SMTP.Enabled {
		fmt.Printf("  SMTP Server:  %s:%d\n", cfg.SMTP.Host, cfg.SMTP.Port)
	}
	if cfg.IMAP.Enabled {
		fmt.Printf("  IMAP Server:  %s:%d\n", cfg.IMAP.Host, cfg.IMAP.Port)
	}
	if cfg.API.Enabled {
		fmt.Printf("  API/Web UI:   http://%s:%d\n", cfg.API.Host, cfg.API.Port)
	}

	fmt.Println()
	fmt.Println("  Press Ctrl+C to stop")
	fmt.Println("=================================================")
}
