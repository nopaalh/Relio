package main

import (
	"os"
	"strings"
	"testing"
)

func TestStatementsKeepQuotedSemicolonsAndRejectMultipleQueries(t *testing.T) {
	got, err := statements("RETURN \"one;two\";\nRETURN 2;\n")
	if err != nil || len(got) != 2 || got[0] != `RETURN "one;two"` {
		t.Fatalf("quoted semicolon split incorrectly: %v", err)
	}
	for _, text := range []string{"RETURN 1; RETURN 2;", "RETURN 1", "RETURN \"unterminated;"} {
		if _, err := statements(text); err == nil {
			t.Fatal("accepted malformed/multiple statement line")
		}
	}
}

func TestConfigPreservesSecretsAndRejectsPlaceholders(t *testing.T) {
	text := "NEO4J_URI=neo4j+s://example.databases.neo4j.io\nNEO4J_USERNAME=neo4j\nNEO4J_PASSWORD='abc#=123'\nNEO4J_DATABASE=neo4j\nRELIO_DATASET_VERSION=p04:test\n"
	c, err := parseConfig(text)
	if err != nil || c.password != "abc#=123" || c.database != "neo4j" {
		t.Fatal("valid credentials were not preserved")
	}
	for _, bad := range []string{strings.ReplaceAll(text, "abc#=123", "PASSWORD_AURA_KAMU"), strings.ReplaceAll(text, "neo4j+s://", "http://"), text + "NEO4J_PASSWORD=duplicate\n"} {
		if _, err := parseConfig(bad); err == nil {
			t.Fatal("accepted unsafe configuration")
		}
	}
}

func TestPreparedFilesProduceTwoSchemaAnd49SeedStatements(t *testing.T) {
	for _, tc := range []struct {
		file  string
		count int
	}{{"schema.cypher", 2}, {"seed-p04.cypher", 49}} {
		data, err := os.ReadFile("../../database/neo4j/" + tc.file)
		if err != nil {
			t.Fatal(err)
		}
		got, err := statements(string(data))
		if err != nil || len(got) != tc.count {
			t.Fatalf("%s: statements=%d err=%v", tc.file, len(got), err)
		}
	}
}
