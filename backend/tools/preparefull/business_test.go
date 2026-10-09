package main

import (
	"encoding/json"
	"github.com/nopaalh/Relio/backend/models"
	"testing"
)

func TestFullSourceRequestorNotDecider(t *testing.T) {
	s, e := readSources("../../../../Datasets", "../../docs/source-audit-snapshot.json")
	if e != nil {
		t.Fatal(e)
	}
	a, e := build(s)
	if e != nil {
		t.Fatal(e)
	}
	proofs := map[string]models.Evidence{}
	for _, r := range a.Records {
		if r.Kind == "evidence" {
			var v models.Evidence
			_ = json.Unmarshal(r.Payload, &v)
			proofs[r.ID] = v
		}
	}
	for _, r := range a.Records {
		if r.Kind != "edges" {
			continue
		}
		var edge models.Edge
		_ = json.Unmarshal(r.Payload, &edge)
		if edge.EdgeType == "DECIDED_BY" {
			for _, id := range edge.EvidenceIDs {
				if proofs[id].SourceField != "diputuskan_oleh" {
					t.Fatal("requestor incorrectly mapped as approver", r.ID)
				}
			}
		}
	}
}
func TestFullSourceDisplayLabelHasFieldProof(t *testing.T) {
	s, e := readSources("../../../../Datasets", "../../docs/source-audit-snapshot.json")
	if e != nil {
		t.Fatal(e)
	}
	a, e := build(s)
	if e != nil {
		t.Fatal(e)
	}
	proofs := map[string]models.Evidence{}
	for _, r := range a.Records {
		if r.Kind == "evidence" {
			var v models.Evidence
			_ = json.Unmarshal(r.Payload, &v)
			proofs[r.ID] = v
		}
	}
	for _, r := range a.Records {
		if r.Kind != "nodes" {
			continue
		}
		var n models.Node
		_ = json.Unmarshal(r.Payload, &n)
		if n.NodeType == "ticket" || n.NodeType == "bug" || n.NodeType == "feature" {
			for _, id := range n.Label.EvidenceIDs {
				f := proofs[id].SourceField
				if f != "judul" && f != "nama" {
					t.Fatal("name supported by an ID field", r.ID, f)
				}
			}
		}
	}
}
