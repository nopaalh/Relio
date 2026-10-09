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

	"github.com/nopaalh/Relio/backend/controllers"
	"github.com/nopaalh/Relio/backend/routes"
	"github.com/nopaalh/Relio/backend/services"
)

func main() {
	addr := os.Getenv("ADDR")
	if addr == "" {
		addr = ":8080"
	}

	healthService := services.NewHealthService()
	healthController := controllers.NewHealthController(healthService)
	// Keep deal data unavailable until the agreed static database adapter is wired.
	dealService := services.NewDealService(nil)
	dealController := controllers.NewDealController(dealService)

	server := &http.Server{
		Addr:              addr,
		Handler:           routes.NewHandler(healthController, dealController),
		ReadHeaderTimeout: 5 * time.Second,
	}

	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	go func() {
		<-ctx.Done()
		shutdownCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()
		if err := server.Shutdown(shutdownCtx); err != nil {
			log.Printf("graceful shutdown failed: %v", err)
		}
	}()

	log.Printf("Relio API listening on %s", addr)
	if err := server.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
		log.Fatal(err)
	}
}
