package main

import "testing"

func TestFullLoadConfigDoesNotTreatSecretAsCode(t *testing.T) {
	c, err := parseConfig("NEO4J_URI=neo4j+s://example.databases.neo4j.io\nNEO4J_USERNAME=neo4j\nNEO4J_PASSWORD='abc#=123'\nNEO4J_DATABASE=neo4j\n")
	if err != nil || c.Password != "abc#=123" {
		t.Fatal("password changed", err)
	}
	if _, err := parseConfig("NEO4J_URI=http://wrong\nNEO4J_USERNAME=x\nNEO4J_PASSWORD=x\nNEO4J_DATABASE=x\n"); err == nil {
		t.Fatal("unsafe URI accepted")
	}
}
func TestFullLoadCapacityStopsBeforeWrites(t *testing.T) {
	if err := checkBudget(100, 200, 600, 400, 500, 1000); err == nil {
		t.Fatal("node limit bypass")
	}
	if err := checkBudget(100, 200, 100, 400, 500, 500); err == nil {
		t.Fatal("relationship limit bypass")
	}
	if err := checkBudget(100, 200, 100, 100, 500, 1000); err != nil {
		t.Fatal(err)
	}
}
