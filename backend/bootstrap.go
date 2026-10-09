package main

import (
	"errors"
	"os"
	"path/filepath"
	"strings"

	"github.com/nopaalh/Relio/backend/repository"
)

var errBootstrapConfig = errors.New("bootstrap configuration unavailable")

type bootstrapConfig struct {
	addr           string
	neo4j          repository.Neo4jConfig
	datasetVersion string
	jevAPIKey      string
}

// Prefer backend/.env from the repo root; never merge a second local file.
func loadLocalEnv(dir string, load func(...string) error) error {
	for _, name := range []string{filepath.Join(dir, "backend", ".env"), filepath.Join(dir, ".env")} {
		err := load(name)
		if errors.Is(err, os.ErrNotExist) {
			continue
		}
		if err != nil {
			return errBootstrapConfig
		}
		return nil
	}
	return nil
}

func readBootstrapConfig(dir string, getenv func(string) string) (bootstrapConfig, error) {
	cfg := bootstrapConfig{
		addr:      getenv("ADDR"),
		jevAPIKey: getenv("TYPESAFE_API_KEY"),
		neo4j: repository.Neo4jConfig{
			URI:      getenv("NEO4J_URI"),
			Username: getenv("NEO4J_USERNAME"),
			Password: getenv("NEO4J_PASSWORD"),
			Database: getenv("NEO4J_DATABASE"),
		},
	}
	if cfg.addr == "" {
		cfg.addr = ":8080"
	}

	version := getenv("RELIO_DATASET_VERSION")
	if version == "" {
		version = fullDatasetVersion
	}
	if strings.TrimSpace(version) != version {
		return cfg, errBootstrapConfig
	}
	cfg.datasetVersion = version
	return cfg, nil
}

func hasNeo4jConfig(cfg repository.Neo4jConfig) bool {
	return strings.TrimSpace(cfg.URI) != "" && strings.TrimSpace(cfg.Username) != "" && cfg.Password != "" && strings.TrimSpace(cfg.Database) != ""
}
