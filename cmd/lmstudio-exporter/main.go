package main

import (
	"context"
	"fmt"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/eXsoR65/lmstudio-exporter/internal/config"
	"github.com/eXsoR65/lmstudio-exporter/internal/lms"
	"github.com/eXsoR65/lmstudio-exporter/internal/metrics"
)

var (
	version   = "dev"
	commit    = "unknown"
	buildDate = "unknown"
)

func main() {
	cfg, err := config.Parse(os.Args[1:], os.Stderr)
	if err != nil {
		os.Exit(2)
	}
	if cfg.ShowVersion {
		fmt.Printf("lmstudio-exporter %s (commit=%s, built=%s)\n", version, commit, buildDate)
		return
	}
	logger := newLogger(cfg.LogLevel, cfg.LogFormat)

	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	store := metrics.NewStore(version, commit, buildDate)
	store.SetLogStreamEnabled(!cfg.DisableLogStream)
	runner := lms.Runner{Path: cfg.LMSPath, Timeout: cfg.CommandTimeout}

	go pollLoop(ctx, logger, runner, store, cfg.PollInterval)
	if !cfg.DisableLogStream {
		streamer := lms.LogStreamer{
			Path:   cfg.LMSPath,
			Logger: logger,
			Callbacks: lms.LogStreamCallbacks{
				OnStats:      store.ObservePrediction,
				OnState:      store.SetLogStreamUp,
				OnParseError: store.IncLogParseError,
				OnEvent:      store.IncLogEvent,
			},
		}
		go streamer.Run(ctx)
	}

	mux := http.NewServeMux()
	mux.Handle(cfg.TelemetryPath, metrics.Handler{Store: store})
	mux.HandleFunc("/healthz", func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "text/plain; charset=utf-8")
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte("ok\n"))
	})
	mux.HandleFunc("/readyz", func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "text/plain; charset=utf-8")
		if !store.Ready() {
			w.WriteHeader(http.StatusServiceUnavailable)
			_, _ = w.Write([]byte("lmstudio unavailable\n"))
			return
		}
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte("ready\n"))
	})
	mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/" {
			http.NotFound(w, r)
			return
		}
		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		_, _ = fmt.Fprintf(w, "<html><body><h1>LM Studio Exporter</h1><p><a href=%q>Metrics</a></p></body></html>\n", cfg.TelemetryPath)
	})

	server := &http.Server{
		Addr:              cfg.ListenAddress,
		Handler:           mux,
		ReadHeaderTimeout: 5 * time.Second,
		IdleTimeout:       60 * time.Second,
	}

	serverErr := make(chan error, 1)
	go func() {
		logger.Info("starting LM Studio Prometheus exporter",
			"version", version,
			"listen_address", cfg.ListenAddress,
			"telemetry_path", cfg.TelemetryPath,
			"log_stream_enabled", !cfg.DisableLogStream,
		)
		if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			serverErr <- err
		}
	}()

	select {
	case <-ctx.Done():
		logger.Info("shutting down")
	case err := <-serverErr:
		logger.Error("HTTP server failed", "error", err)
		stop()
	}

	shutdownCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	if err := server.Shutdown(shutdownCtx); err != nil {
		logger.Error("HTTP server shutdown failed", "error", err)
	}
}

func pollLoop(ctx context.Context, logger *slog.Logger, runner lms.Runner, store *metrics.Store, interval time.Duration) {
	poll := func() {
		status, err := runner.DaemonStatus(ctx)
		if err != nil {
			store.SetDaemonOnly(lms.DaemonStatus{}, false)
			logger.Debug("LM Studio daemon status poll failed", "error", err)
			return
		}
		if !status.Running {
			store.SetDaemonOnly(status, true)
			return
		}
		models, err := runner.Models(ctx)
		if err != nil {
			store.SetDaemonOnly(status, false)
			logger.Warn("lms ps poll failed", "error", err)
			return
		}
		store.SetPoll(status, models, true)
	}

	poll()
	ticker := time.NewTicker(interval)
	defer ticker.Stop()
	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			poll()
		}
	}
}

func newLogger(level, format string) *slog.Logger {
	var lvl slog.Level
	switch level {
	case "debug":
		lvl = slog.LevelDebug
	case "warn":
		lvl = slog.LevelWarn
	case "error":
		lvl = slog.LevelError
	default:
		lvl = slog.LevelInfo
	}
	opts := &slog.HandlerOptions{Level: lvl}
	if format == "json" {
		return slog.New(slog.NewJSONHandler(os.Stderr, opts))
	}
	return slog.New(slog.NewTextHandler(os.Stderr, opts))
}
