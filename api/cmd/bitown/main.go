package main

import (
	"context"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"strings"
	"syscall"
	"time"

	"github.com/interestic/bitown/internal/city"
	"github.com/interestic/bitown/internal/middleware"
	"github.com/interestic/bitown/internal/store"

	"github.com/go-chi/chi/v5"
	chimw "github.com/go-chi/chi/v5/middleware"
	"github.com/joho/godotenv"
)

func main() {
	_ = godotenv.Load()

	log := slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelDebug}))
	slog.SetDefault(log)

	dbURL := mustEnv("DATABASE_URL")
	redisURL := mustEnv("REDIS_URL")
	saltSeed := mustEnv("DAILY_SALT_SEED")

	ctx := context.Background()

	db, err := store.NewPostgres(ctx, dbURL)
	if err != nil {
		slog.Error("postgres connect failed", "err", err)
		os.Exit(1)
	}
	defer db.Close()

	rdb, err := store.NewRedis(redisURL)
	if err != nil {
		slog.Error("redis connect failed", "err", err)
		os.Exit(1)
	}
	defer func() {
		if err := rdb.Close(); err != nil {
			slog.Warn("redis close error", "err", err)
		}
	}()

	slog.Info("stores connected")

	r := chi.NewRouter()
	r.Use(chimw.RequestID)
	r.Use(middleware.ClientIP)
	r.Use(chimw.Logger)
	r.Use(chimw.Recoverer)
	r.Use(middleware.CORS)

	cityHandler := city.NewHandler(db, rdb, saltSeed)

	embedDir := os.Getenv("BITOWN_EMBED_DIR")
	if embedDir == "" {
		embedDir = "../embed"
	}
	if st, err := os.Stat(embedDir); err == nil && st.IsDir() {
		r.Handle("/embed/*", embedFileServer(embedDir))
		slog.Info("embed widget enabled", "dir", embedDir)
	} else {
		slog.Warn("embed widget disabled: set BITOWN_EMBED_DIR to the embed/ directory", "dir", embedDir, "err", err)
	}

	r.Get("/api/health", handleHealth)
	r.Route("/api/cities", func(r chi.Router) {
		r.Post("/", cityHandler.Create)
		r.Get("/{slug}", cityHandler.Get)
		r.Get("/{slug}/map.png", cityHandler.MapPNG)
		r.Get("/{slug}/events", cityHandler.Events)
		r.Post("/{slug}/support", cityHandler.Support)
	})
	if city.DebugModeEnabled() {
		r.Get("/api/debug/cities/{slug}", cityHandler.DebugGet)
		slog.Info("debug mode enabled",
			"route", "/api/debug/cities/{slug}",
			"map_query", "pop,ind,tra,sec,env,com on /api/cities/{slug}/map.png",
		)
	}
	r.Get("/api/rankings", cityHandler.Rankings)
	r.Get("/badge/{slug}.svg", cityHandler.Badge)

	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}

	srv := &http.Server{
		Addr:              ":" + port,
		Handler:           r,
		ReadHeaderTimeout: 5 * time.Second,
		ReadTimeout:       10 * time.Second,
		WriteTimeout:      10 * time.Second,
		IdleTimeout:       60 * time.Second,
	}

	done := make(chan struct{})
	go func() {
		quit := make(chan os.Signal, 1)
		signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
		<-quit
		slog.Info("shutting down")
		ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()
		_ = srv.Shutdown(ctx)
		close(done)
	}()

	slog.Info("bitown API started", "addr", srv.Addr)
	if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
		slog.Error("listen failed", "err", err)
		os.Exit(1)
	}
	<-done
}

func handleHealth(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	_, _ = w.Write([]byte(`{"status":"ok"}`))
}

// embedFileServer serves static files from embedDir at /embed/*.
// chi Mount leaves r.URL.Path unchanged, which breaks net/http FileServer.
func embedFileServer(embedDir string) http.Handler {
	return http.StripPrefix("/embed", noDirListing(http.FileServer(http.Dir(embedDir))))
}

// noDirListing wraps a FileServer so directory URLs return 404 instead of listings.
func noDirListing(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if strings.HasSuffix(r.URL.Path, "/") {
			http.NotFound(w, r)
			return
		}
		next.ServeHTTP(w, r)
	})
}

func mustEnv(key string) string {
	v := os.Getenv(key)
	if v == "" {
		slog.Error("required env var missing", "key", key)
		os.Exit(1)
	}
	return v
}
