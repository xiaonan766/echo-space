package main

import (
	"context"
	"errors"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/xiaonan766/echo-space/echo-space-backend/internal/app"
	"github.com/xiaonan766/echo-space/echo-space-backend/internal/config"
)

func main() {
	cfg := config.Load()

	application, err := app.New(context.Background(), cfg)
	if err != nil {
		log.Fatalf("start application: %v", err)
	}
	defer application.Close()

	server := &http.Server{
		Addr:    cfg.Server.Address(),
		Handler: application.Router(),
	}

	go func() {
		log.Printf("echo-space backend listening on %s", cfg.Server.Address())
		if err := server.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			log.Fatalf("listen and serve: %v", err)
		}
	}()

	quit := make(chan os.Signal, 1)
	signal.Notify(quit, os.Interrupt, syscall.SIGTERM)
	<-quit

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	if err := server.Shutdown(ctx); err != nil {
		log.Printf("shutdown server: %v", err)
	}
}
