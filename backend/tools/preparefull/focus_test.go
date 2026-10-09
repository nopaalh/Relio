package main

import (
	"encoding/json"
	"github.com/nopaalh/Relio/backend/models"
	"testing"
)

func TestFullSourceEveryEventHasFocusNode(t *testing.T) {
	s, e := readSources("../../../../Datasets", "../../docs/source-audit-snapshot.json")
	if e != nil {
		t.Fatal(e)
	}
	a, e := build(s)
	if e != nil {
		t.Fatal(e)
	}
	nodes := map[string]bool{}
	for _, r := range a.Records {
		if r.Kind == "nodes" {
			nodes[r.ID] = true
		}
	}
	for _, r := range a.Records {
		if r.Kind != "events" {
			continue
		}
		var event models.Event
		_ = json.Unmarshal(r.Payload, &event)
		if !nodes[event.EventID] {
			t.Fatal("timeline event cannot be retained as graph focus", event.EventID)
		}
	}
}
