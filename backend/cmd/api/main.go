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

	"finos.com/api/internal/auth"
	"finos.com/api/internal/config"
	"finos.com/api/internal/database"
	"finos.com/api/internal/server"
	"finos.com/api/internal/user"
)

func main() {
	cfg, err := config.Load()
	if err != nil {
		log.Fatalf("load configuration: %v", err)
	}

	appContext, stop := signal.NotifyContext(
		context.Background(),
		os.Interrupt,
		syscall.SIGTERM,
	)
	defer stop()

	databasePool, err := database.Open(appContext, cfg.DatabaseURL)
	if err != nil {
		log.Fatalf("connect to database: %v", err)
	}
	defer databasePool.Close()

	tokenManager, err := auth.NewTokenManager(cfg.JWTSecret, cfg.JWTTTL)
	if err != nil {
		log.Fatalf("configure JWT manager: %v", err)
	}

	userRepository := user.NewRepository(databasePool)
	userService := user.NewService(userRepository)
	userHandler := user.NewHandler(userService, tokenManager)

	httpServer := &http.Server{
		Addr:              ":" + cfg.HTTPPort,
		Handler:           server.New(databasePool, userHandler),
		ReadHeaderTimeout: 5 * time.Second,
		ReadTimeout:       10 * time.Second,
		WriteTimeout:      10 * time.Second,
		IdleTimeout:       60 * time.Second,
	}

	go func() {
		<-appContext.Done()

		log.Println("shutting down HTTP server")

		shutdownContext, cancel := context.WithTimeout(
			context.Background(),
			10*time.Second,
		)
		defer cancel()

		if err := httpServer.Shutdown(shutdownContext); err != nil {
			log.Printf("HTTP server shutdown error: %v", err)
		}
	}()

	log.Printf(
		"starting API environment=%s address=%s",
		cfg.AppEnv,
		httpServer.Addr,
	)

	if err := httpServer.ListenAndServe(); err != nil &&
		!errors.Is(err, http.ErrServerClosed) {
		log.Fatalf("HTTP server failed: %v", err)
	}
}
