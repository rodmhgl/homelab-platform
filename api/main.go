package main

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	"github.com/kelseyhightower/envconfig"
	"github.com/rodmhgl/homelab-platform/api/internal/argocd"
	"github.com/rodmhgl/homelab-platform/api/internal/compliance"
	"github.com/rodmhgl/homelab-platform/api/internal/infra"
	"github.com/rodmhgl/homelab-platform/api/internal/scaffold"
	"github.com/rodmhgl/homelab-platform/api/internal/secrets"
	"github.com/rodmhgl/homelab-platform/api/internal/webhooks"
)

// Config holds the application configuration loaded from environment variables
type Config struct {
	Port            int    `envconfig:"PORT" default:"8080"`
	LogLevel        string `envconfig:"LOG_LEVEL" default:"info"`
	ShutdownTimeout int    `envconfig:"SHUTDOWN_TIMEOUT" default:"30"`

	// Kubernetes API configuration
	KubeConfig string `envconfig:"KUBECONFIG"`
	InCluster  bool   `envconfig:"IN_CLUSTER" default:"true"`

	// Argo CD configuration
	ArgocdServerURL string `envconfig:"ARGOCD_SERVER_URL" required:"true"`
	ArgocdToken     string `envconfig:"ARGOCD_TOKEN" required:"true"`

	// GitHub configuration (for GitOps commits)
	GithubToken  string `envconfig:"GITHUB_TOKEN" required:"true"`
	GithubOrg    string `envconfig:"GITHUB_ORG" required:"true"`
	PlatformRepo string `envconfig:"PLATFORM_REPO" default:"homelab-platform"`

	// AI Operations configuration
	OpenAIAPIKey    string `envconfig:"OPENAI_API_KEY"`
	HolmesGPTURL    string `envconfig:"HOLMESGPT_URL"`
	KAgentNamespace string `envconfig:"KAGENT_NAMESPACE" default:"kagent-system"`

	// Scaffold configuration
	ScaffoldTemplates string `envconfig:"SCAFFOLD_TEMPLATES" default:"/app/scaffolds"`
	ScaffoldWorkDir   string `envconfig:"SCAFFOLD_WORK_DIR" default:"/tmp/scaffold"`
}

func main() {
	// Load configuration from environment
	var cfg Config
	if err := envconfig.Process("", &cfg); err != nil {
		fmt.Fprintf(os.Stderr, "Failed to load configuration: %v\n", err)
		os.Exit(1)
	}

	// Setup structured logging
	logLevel := parseLogLevel(cfg.LogLevel)
	logger := slog.New(slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{
		Level: logLevel,
	}))
	slog.SetDefault(logger)

	slog.Info("Starting Platform API",
		"port", cfg.Port,
		"log_level", cfg.LogLevel,
	)

	// Initialize scaffold handler
	scaffoldHandler, err := scaffold.NewHandler(&scaffold.Config{
		GithubToken:       cfg.GithubToken,
		GithubOrg:         cfg.GithubOrg,
		PlatformRepo:      cfg.PlatformRepo,
		ScaffoldTemplates: cfg.ScaffoldTemplates,
		WorkDir:           cfg.ScaffoldWorkDir,
	})
	if err != nil {
		slog.Error("Failed to initialize scaffold handler", "error", err)
		os.Exit(1)
	}

	// Initialize Argo CD handler
	argocdHandler := argocd.NewHandler(&argocd.Config{
		ServerURL: cfg.ArgocdServerURL,
		Token:     cfg.ArgocdToken,
	})

	// Initialize event store (circular buffer, max 1000 events)
	eventStore := compliance.NewInMemoryEventStore(1000)

	// Initialize compliance handler
	complianceHandler, err := compliance.NewHandler(&compliance.Config{
		KubeConfig: cfg.KubeConfig,
		InCluster:  cfg.InCluster,
	}, eventStore)
	if err != nil {
		slog.Error("Failed to initialize compliance handler", "error", err)
		os.Exit(1)
	}

	// Initialize webhook handler
	webhookHandler := webhooks.NewHandler(eventStore)

	// Initialize infra handler
	infraHandler, err := infra.NewHandler(&infra.Config{
		KubeConfig: cfg.KubeConfig,
		InCluster:  cfg.InCluster,
	}, cfg.GithubToken)
	if err != nil {
		slog.Error("Failed to initialize infra handler", "error", err)
		os.Exit(1)
	}

	// Initialize secrets handler
	secretsHandler, err := secrets.NewHandler(&secrets.Config{
		KubeConfig: cfg.KubeConfig,
		InCluster:  cfg.InCluster,
	})
	if err != nil {
		slog.Error("Failed to initialize secrets handler", "error", err)
		os.Exit(1)
	}

	// Initialize router
	r := setupRouter(scaffoldHandler, argocdHandler, complianceHandler, infraHandler, secretsHandler, webhookHandler)

	// Create HTTP server
	srv := &http.Server{
		Addr:         fmt.Sprintf(":%d", cfg.Port),
		Handler:      r,
		ReadTimeout:  15 * time.Second,
		WriteTimeout: 15 * time.Second,
		IdleTimeout:  60 * time.Second,
	}

	// Start server in goroutine
	serverErrors := make(chan error, 1)
	go func() {
		slog.Info("API server listening", "addr", srv.Addr)
		serverErrors <- srv.ListenAndServe()
	}()

	// Setup graceful shutdown
	shutdown := make(chan os.Signal, 1)
	signal.Notify(shutdown, os.Interrupt, syscall.SIGTERM)

	// Block until shutdown signal or server error
	select {
	case err := <-serverErrors:
		if err != nil && !errors.Is(err, http.ErrServerClosed) {
			slog.Error("Server error", "error", err)
			os.Exit(1)
		}
	case sig := <-shutdown:
		slog.Info("Shutdown signal received", "signal", sig)

		// Create shutdown context with timeout
		ctx, cancel := context.WithTimeout(context.Background(), time.Duration(cfg.ShutdownTimeout)*time.Second)
		defer cancel()

		// Attempt graceful shutdown
		if err := srv.Shutdown(ctx); err != nil {
			slog.Error("Graceful shutdown failed, forcing", "error", err)
			if err := srv.Close(); err != nil {
				slog.Error("Force close failed", "error", err)
			}
		}
		slog.Info("Shutdown complete")
	}
}

// setupRouter configures the Chi router with middleware and routes
func setupRouter(scaffoldHandler *scaffold.Handler, argocdHandler *argocd.Handler, complianceHandler *compliance.Handler, infraHandler *infra.Handler, secretsHandler *secrets.Handler, webhookHandler *webhooks.Handler) *chi.Mux {
	r := chi.NewRouter()

	// Global middleware
	r.Use(middleware.RequestID)
	r.Use(middleware.RealIP)
	r.Use(loggerMiddleware)
	r.Use(middleware.Recoverer)
	r.Use(middleware.Timeout(60 * time.Second))

	// Health and readiness endpoints (no auth required)
	r.Get("/health", healthHandler)
	r.Get("/ready", readyHandler)

	// API v1 routes
	r.Route("/api/v1", func(r chi.Router) {
		// Authentication middleware for all API routes
		r.Use(authMiddleware)

		// Scaffold endpoints
		r.Route("/scaffold", func(r chi.Router) {
			r.Post("/", scaffoldHandler.HandleCreate)
		})

		// Application management endpoints
		r.Route("/apps", func(r chi.Router) {
			r.Get("/", argocdHandler.HandleListApps)
			r.Route("/{name}", func(r chi.Router) {
				r.Get("/", argocdHandler.HandleGetApp)
				r.Post("/sync", argocdHandler.HandleSyncApp)
			})
		})

		// Infrastructure endpoints
		r.Route("/infra", func(r chi.Router) {
			r.Get("/", infraHandler.HandleListAllClaims)
			r.Post("/", infraHandler.HandleCreateClaim)
			r.Get("/storage", infraHandler.HandleListStorageClaims)
			r.Get("/vaults", infraHandler.HandleListVaultClaims)
			r.Route("/{kind}/{name}", func(r chi.Router) {
				r.Get("/", infraHandler.HandleGetResource)
				r.Delete("/", infraHandler.HandleDeleteClaim)
			})
		})

		// Compliance endpoints
		r.Route("/compliance", func(r chi.Router) {
			r.Get("/summary", complianceHandler.HandleSummary)
			r.Get("/policies", complianceHandler.HandlePolicies)
			r.Get("/violations", complianceHandler.HandleViolations)
			r.Get("/vulnerabilities", complianceHandler.HandleVulnerabilities)
			r.Get("/events", complianceHandler.HandleEvents)
		})

		// Secrets endpoints
		r.Route("/secrets", func(r chi.Router) {
			r.Get("/{namespace}", secretsHandler.HandleListSecrets)
		})

		// Investigation endpoints (HolmesGPT)
		r.Route("/investigate", func(r chi.Router) {
			r.Post("/", notImplementedHandler("POST /api/v1/investigate"))
			r.Get("/{id}", notImplementedHandler("GET /api/v1/investigate/{id}"))
		})

		// AI Agent endpoints (kagent)
		r.Route("/agent", func(r chi.Router) {
			r.Post("/ask", notImplementedHandler("POST /api/v1/agent/ask"))
		})

		// Webhook endpoints (no auth middleware - validated by payload)
		r.Route("/webhooks", func(r chi.Router) {
			r.Post("/falco", webhookHandler.HandleFalcoWebhook)
			r.Post("/argocd", notImplementedHandler("POST /api/v1/webhooks/argocd"))
		})
	})

	return r
}

// loggerMiddleware logs HTTP requests with structured logging
func loggerMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()

		ww := middleware.NewWrapResponseWriter(w, r.ProtoMajor)
		next.ServeHTTP(ww, r)

		slog.Info("HTTP request",
			"method", r.Method,
			"path", r.URL.Path,
			"status", ww.Status(),
			"bytes", ww.BytesWritten(),
			"duration_ms", time.Since(start).Milliseconds(),
			"request_id", middleware.GetReqID(r.Context()),
			"remote_addr", r.RemoteAddr,
		)
	})
}

// authMiddleware validates Bearer token authentication
func authMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Skip webhook endpoints (they use payload validation instead)
		if r.URL.Path == "/api/v1/webhooks/falco" || r.URL.Path == "/api/v1/webhooks/argocd" {
			next.ServeHTTP(w, r)
			return
		}

		authHeader := r.Header.Get("Authorization")
		if authHeader == "" {
			http.Error(w, `{"error":"missing authorization header"}`, http.StatusUnauthorized)
			return
		}

		// TODO: Implement actual token validation
		// For now, just check that a Bearer token is present
		if len(authHeader) < 7 || authHeader[:7] != "Bearer " {
			http.Error(w, `{"error":"invalid authorization format"}`, http.StatusUnauthorized)
			return
		}

		next.ServeHTTP(w, r)
	})
}

// healthHandler returns 200 OK if the service is running
func healthHandler(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	w.Write([]byte(`{"status":"ok"}`))
}

// readyHandler returns 200 OK if the service is ready to accept traffic
func readyHandler(w http.ResponseWriter, r *http.Request) {
	// TODO: Add actual readiness checks (K8s API connectivity, Argo CD API, etc.)
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	w.Write([]byte(`{"status":"ready"}`))
}

// notImplementedHandler returns a 501 Not Implemented response
func notImplementedHandler(endpoint string) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusNotImplemented)
		w.Write([]byte(fmt.Sprintf(`{"error":"endpoint not yet implemented","endpoint":"%s"}`, endpoint)))
	}
}

// parseLogLevel converts string log level to slog.Level
func parseLogLevel(level string) slog.Level {
	switch level {
	case "debug":
		return slog.LevelDebug
	case "info":
		return slog.LevelInfo
	case "warn":
		return slog.LevelWarn
	case "error":
		return slog.LevelError
	default:
		return slog.LevelInfo
	}
}
