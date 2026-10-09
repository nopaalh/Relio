package main

import "testing"

func TestFullLoadPartialNamespaceStillNeedsCapacity(t *testing.T) {
	nodes, links, e := remainingNamespaceSize(1000, 2000, 200, 300)
	if e != nil || nodes != 800 || links != 1700 {
		t.Fatal("partial namespace must reserve remaining nodes/links")
	}
	if e = checkBudget(400, 500, nodes, links, 1000, 2000); e == nil {
		t.Fatal("partial retry capacity bypass")
	}
	if _, _, e = remainingNamespaceSize(100, 200, 101, 100); e == nil {
		t.Fatal("unexpected namespace size accepted")
	}
}

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
