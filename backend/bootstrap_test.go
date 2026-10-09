package main

import (
	"errors"
	"os"
	"path/filepath"
	"testing"

	"github.com/joho/godotenv"
	"github.com/nopaalh/Relio/backend/repository"
)

func TestMain(m *testing.M) {
	if os.Setenv("RUN_JEV_LIVE", "0") != nil || os.Setenv("RELIO_NEO4J_INTEGRATION", "0") != nil {
		os.Exit(1)
	}
	os.Exit(m.Run())
}

func writeBootstrapFixture(t *testing.T, dir, name, content string) {
	t.Helper()
	path := filepath.Join(dir, name)
	if err := os.MkdirAll(filepath.Dir(path), 0700); err != nil {
		t.Fatal("could not create fixture directory")
	}
	if err := os.WriteFile(path, []byte(content), 0600); err != nil {
		t.Fatal("could not write dummy fixture")
	}
}

// Model Load's non-overriding behavior using dummy files and a process-env map,
// without reading or modifying real configuration variables.
func bootstrapMapLoader(env map[string]string) func(...string) error {
	return func(names ...string) error {
		values, err := godotenv.Read(names...)
		if err != nil {
			return err
		}
		for key, value := range values {
			if _, exists := env[key]; !exists {
				env[key] = value
			}
		}
		return nil
	}
}

func TestBootstrapLocalEnvPathsAndPrecedence(t *testing.T) {
	for _, tc := range []struct {
		name string
		file string
	}{
		{"backend cwd", ".env"},
		{"repo root cwd", filepath.Join("backend", ".env")},
	} {
		t.Run(tc.name, func(t *testing.T) {
			dir := t.TempDir()
			content := "ADDR=file-address\nTYPESAFE_API_KEY=dummy-file-key\nRELIO_DATASET_VERSION=fixture:published\n"
			writeBootstrapFixture(t, dir, tc.file, content)
			if tc.name == "repo root cwd" {
				writeBootstrapFixture(t, dir, ".env", "ROOT_ONLY=must-not-load\n")
			}
			env := map[string]string{"ADDR": "process-address", "TYPESAFE_API_KEY": ""}
			if err := loadLocalEnv(dir, bootstrapMapLoader(env)); err != nil {
				t.Fatal("dummy environment did not load")
			}
			cfg, err := readBootstrapConfig(dir, func(key string) string { return env[key] })
			if err != nil {
				t.Fatal("dummy config did not resolve")
			}
			if cfg.addr != "process-address" || cfg.jevAPIKey != "" || cfg.datasetVersion != "fixture:published" {
				t.Fatal("process environment precedence was not preserved")
			}
			if _, exists := env["ROOT_ONLY"]; exists {
				t.Fatal("loader merged a second local environment file")
			}
			data, err := os.ReadFile(filepath.Join(dir, tc.file))
			if err != nil || string(data) != content {
				t.Fatal("loader modified the dummy environment file")
			}
		})
	}
}

func TestBootstrapLocalEnvMissingIsOptional(t *testing.T) {
	if err := loadLocalEnv(t.TempDir(), bootstrapMapLoader(map[string]string{})); err != nil {
		t.Fatal("missing local environment should permit process-only configuration")
	}
}

func TestBootstrapLocalEnvErrorsAreGeneric(t *testing.T) {
	calls := 0
	err := loadLocalEnv(t.TempDir(), func(...string) error {
		calls++
		return errors.New("DUMMY_SECRET parser diagnostic")
	})
	if !errors.Is(err, errBootstrapConfig) || err.Error() != "bootstrap configuration unavailable" || calls != 1 {
		t.Fatal("loader must redact errors and not fall through a broken file")
	}

	dir := t.TempDir()
	writeBootstrapFixture(t, dir, filepath.Join("backend", ".env"), "BROKEN=\"unterminated-dummy-value\n")
	writeBootstrapFixture(t, dir, ".env", "ADDR=must-not-load\n")
	if err := loadLocalEnv(dir, bootstrapMapLoader(map[string]string{})); !errors.Is(err, errBootstrapConfig) {
		t.Fatal("malformed dummy environment should fail safely")
	}
}

func TestBootstrapConfigUsesOnlyInjectedLookup(t *testing.T) {
	dir := t.TempDir()
	writeBootstrapFixture(t, dir, filepath.Join("backend", "database", "neo4j", "manifest-p04.json"), "not-json")
	env := map[string]string{
		"ADDR":                  "127.0.0.1:9000",
		"TYPESAFE_API_KEY":      "dummy-runtime-key",
		"NEO4J_URI":             "neo4j+s://bootstrap.invalid",
		"NEO4J_USERNAME":        "dummy-user",
		"NEO4J_PASSWORD":        "dummy-password",
		"NEO4J_DATABASE":        "dummy-database",
		"RELIO_DATASET_VERSION": "fixture:from-process",
	}
	lookups := map[string]bool{}
	cfg, err := readBootstrapConfig(dir, func(key string) string {
		if _, allowed := env[key]; !allowed {
			t.Fatalf("unexpected config lookup name: %s", key)
		}
		lookups[key] = true
		return env[key]
	})
	if err != nil || len(lookups) != len(env) {
		t.Fatal("config did not resolve only the declared lookup names")
	}
	if cfg.addr != env["ADDR"] || cfg.datasetVersion != env["RELIO_DATASET_VERSION"] || cfg.jevAPIKey != env["TYPESAFE_API_KEY"] {
		t.Fatal("explicit lookup values were not preserved")
	}
	if cfg.neo4j.URI != env["NEO4J_URI"] || cfg.neo4j.Username != env["NEO4J_USERNAME"] || cfg.neo4j.Password != env["NEO4J_PASSWORD"] || cfg.neo4j.Database != env["NEO4J_DATABASE"] {
		t.Fatal("Neo4j config did not use the injected lookup")
	}
	if !hasNeo4jConfig(cfg.neo4j) {
		t.Fatal("complete dummy Neo4j config was not recognized")
	}
}

func TestBootstrapConfigDefaultsToPublishedFullDataset(t *testing.T) {
	cfg, err := readBootstrapConfig(t.TempDir(), func(string) string { return "" })
	if err != nil || cfg.datasetVersion != fullDatasetVersion {
		t.Fatal("bootstrap should default to the published full dataset")
	}
	if cfg.addr != ":8080" || cfg.jevAPIKey != "" || hasNeo4jConfig(cfg.neo4j) {
		t.Fatal("missing config did not retain safe defaults")
	}
}

func TestBootstrapConfigRejectsInvalidExplicitVersion(t *testing.T) {
	cfg, err := readBootstrapConfig(t.TempDir(), func(key string) string {
		if key == "RELIO_DATASET_VERSION" {
			return " fixture:invalid "
		}
		return ""
	})
	if !errors.Is(err, errBootstrapConfig) || cfg.datasetVersion != "" {
		t.Fatal("invalid explicit version must not silently fall back")
	}
}

func TestBootstrapNeo4jRequiresAllFields(t *testing.T) {
	complete := repository.Neo4jConfig{URI: "neo4j+s://bootstrap.invalid", Username: "dummy-user", Password: "dummy-password", Database: "dummy-database"}
	for _, field := range []string{"uri", "username", "password", "database"} {
		t.Run(field, func(t *testing.T) {
			cfg := complete
			switch field {
			case "uri":
				cfg.URI = ""
			case "username":
				cfg.Username = " "
			case "password":
				cfg.Password = ""
			case "database":
				cfg.Database = " "
			}
			if hasNeo4jConfig(cfg) {
				t.Fatal("partial Neo4j config must not initiate a connection")
			}
		})
	}
}
