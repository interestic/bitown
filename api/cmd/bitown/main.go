package main

import (
	"context"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
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
	defer rdb.Close()

	slog.Info("stores connected")

	r := chi.NewRouter()
	r.Use(chimw.RequestID)
	r.Use(chimw.RealIP)
	r.Use(chimw.Logger)
	r.Use(chimw.Recoverer)
	r.Use(middleware.CORS)

	cityHandler := city.NewHandler(db, rdb, saltSeed)

	r.Get("/api/health", handleHealth)
	r.Route("/api/cities", func(r chi.Router) {
		r.Post("/", cityHandler.Create)
		r.Get("/{slug}", cityHandler.Get)
		r.Post("/{slug}/support", cityHandler.Support)
	})
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

func mustEnv(key string) string {
	v := os.Getenv(key)
	if v == "" {
		slog.Error("required env var missing", "key", key)
		os.Exit(1)
	}
	return v
}
