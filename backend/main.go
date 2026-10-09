package main

import (
	"context"
	"errors"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/signal"
	"strings"
	"syscall"
	"time"

	"github.com/joho/godotenv"
	"github.com/nopaalh/Relio/backend/controllers"
	"github.com/nopaalh/Relio/backend/models"
	"github.com/nopaalh/Relio/backend/repository"
	"github.com/nopaalh/Relio/backend/routes"
	"github.com/nopaalh/Relio/backend/services"
)

func main() {
	if err := run(); err != nil {
		log.Fatal("Relio API stopped unexpectedly")
	}
}

const fullDatasetVersion = "full:d8e235c9a52b20fc6a16711f292744f3a06ff52096297ca3805e9f99dcb05a40"

func demoAccessScope(datasetVersion string) models.AccessScope {
	if datasetVersion != fullDatasetVersion {
		return models.AccessScope{AllowedAccountIDs: []string{"P04"}, AllowedDealIDs: []string{"DL-004"}}
	}
	accounts := make([]string, 22)
	deals := make([]string, 22)
	for i := 1; i <= 22; i++ {
		accounts[i-1] = fmt.Sprintf("P%02d", i)
		deals[i-1] = fmt.Sprintf("DL-%03d", i)
	}
	return models.AccessScope{
		AllowedAccountIDs: accounts, AllowedDealIDs: deals,
		AllowedAnalogAccountIDs: []string{"C01", "C23", "C24"},
		AllowCompanyEvidence:    true,
	}
}

func run() error {
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	if err := loadLocalEnv(".", godotenv.Load); err != nil {
		log.Print("Local environment configuration unavailable; using process environment")
	}
	cfg, configErr := readBootstrapConfig(".", os.Getenv)
	if configErr != nil {
		log.Print("Dataset configuration unavailable; context data unavailable")
	}

	var reader services.ContextRepository
	if configErr == nil && hasNeo4jConfig(cfg.neo4j) {
		startupCtx, cancel := context.WithTimeout(ctx, 15*time.Second)
		var db interface {
			services.ContextRepository
			Close(context.Context) error
		}
		var err error
		if strings.HasPrefix(cfg.datasetVersion, "full:") {
			db, err = repository.NewNeo4jFullContextRepository(startupCtx, cfg.neo4j)
		} else {
			db, err = repository.NewNeo4jContextRepository(startupCtx, cfg.neo4j)
		}
		cancel()
		if err != nil || db == nil {
			log.Print("Neo4j unavailable; context data unavailable")
		} else {
			reader = db
			defer func() {
				closeCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
				defer cancel()
				if err := db.Close(closeCtx); err != nil {
					log.Print("Neo4j shutdown failed")
				}
			}()
		}
	} else {
		log.Print("Neo4j not initialized; context data unavailable")
	}

	var jev services.JEVEvaluator
	if cfg.jevAPIKey != "" {
		client, err := repository.NewJEVClient(repository.JEVClientConfig{APIKey: cfg.jevAPIKey})
		if err != nil {
			log.Print("JEV configuration unavailable; scoring unavailable")
		} else {
			jev = client
		}
	}

	access := demoAccessScope(cfg.datasetVersion)
	contextService := services.NewContextService(reader, cfg.datasetVersion, access, jev)
	healthController := controllers.NewHealthController(services.NewHealthService())
	server := &http.Server{
		Addr:              cfg.addr,
		Handler:           routes.NewContextHandler(healthController, controllers.NewContextController(contextService)),
		ReadHeaderTimeout: 5 * time.Second,
	}

	shutdownDone := make(chan struct{})
	go func() {
		defer close(shutdownDone)
		<-ctx.Done()
		shutdownCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()
		if err := server.Shutdown(shutdownCtx); err != nil {
			log.Print("HTTP shutdown failed; closing active connections")
			_ = server.Close()
		}
	}()

	log.Print("Relio API starting")
	err := server.ListenAndServe()
	stop()
	<-shutdownDone
	if err != nil && !errors.Is(err, http.ErrServerClosed) {
		return errors.New("HTTP server unavailable")
	}
	return nil
}
