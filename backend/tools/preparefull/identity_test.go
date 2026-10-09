package main

import (
	"github.com/nopaalh/Relio/backend/models"
	"testing"
)

func TestFullSourceEmailSnapshotDoesNotInventHistoricalIdentity(t *testing.T) {
	b := &builder{src: sources{rows: map[string][]map[string]string{"crm_contacts.csv": {{"email": "illustration@example.test", "contact_id": "OTHER", "account_id_saat_ini": "FOREIGN"}}}}, nodes: map[string]models.Node{}, scope: map[string]map[string]bool{}, from: map[string]models.Date{}}
	p := b.emailParticipant("illustration@example.test", "PRIMARY", "proof", "2026-09-01")
	if p.NodeID != nil || len(p.CandidateNodeIDs) != 0 || len(b.nodes) != 0 {
		t.Fatal("current/foreign email match leaked an invented historical native association")
	}
	if p.RawRef == nil || *p.RawRef != "illustration@example.test" {
		t.Fatal("lost raw source identity")
	}
}
