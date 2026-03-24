package main

import (
	"context"
	"errors"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/DrummDaddy/task_service/internal/app"
	"github.com/DrummDaddy/task_service/internal/config"
	"go.uber.org/zap"
)

func main() {
	cfg, err := config.Load()
	if err != nil {
		panic(err)
	}
	logger := app.NewLogger(cfg.Env)
	defer func() { _ = logger.Sync() }()
	srv, err := app.NewServer(cfg, logger)
	if err != nil {
		logger.Fatal("server init failed", zap.Error(err))
	}

	httpSrv := &http.Server{
		Addr:              cfg.HTTP.Addr,
		Handler:           srv.Router,
		ReadTimeout:       15 * time.Second,
		ReadHeaderTimeout: 10 * time.Second,
		WriteTimeout:      15 * time.Second,
		IdleTimeout:       60 * time.Second,
	}
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	go func() {
		logger.Info("http server listening", app.ZapString("address", cfg.HTTP.Addr))
		if err := httpSrv.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			logger.Fatal("http server listen failed", zap.Error(err))
		}
	}()

	<-ctx.Done()
	logger.Info("shutting down http server...")

	shutdownCtx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	_ = httpSrv.Shutdown(shutdownCtx)
	_ = srv.Close(shutdownCtx)

	logger.Info("shutdown http server")
}
