package main

import (
	"encoding/json"
	"github.com/nopaalh/Relio/backend/models"
	"github.com/nopaalh/Relio/backend/repository"
	"testing"
)

func TestFullSourceRealGraphAndNativeJoins(t *testing.T) {
	s, err := readSources("../../../../Datasets", "../../docs/source-audit-snapshot.json")
	if err != nil {
		t.Fatal(err)
	}
	a, err := build(s)
	if err != nil {
		t.Fatal(err)
	}
	if len(a.Manifest.Sources) != 15 || len(a.Manifest.DealIDs) != 22 || len(a.Manifest.AccountIDs) != 45 {
		t.Fatal("coverage lost")
	}
	var first []byte
	first, _ = json.Marshal(a)
	again, err := build(s)
	if err != nil {
		t.Fatal(err)
	}
	second, _ := json.Marshal(again)
	if string(first) != string(second) {
		t.Fatal("nondeterministic preparation")
	}
	count := 0
	for _, n := range a.Manifest.SourceCounts {
		count += n
	}
	if count != 229627 {
		t.Fatal("source rows not accounted")
	}
	interactions := map[string]int{}
	decisions := map[string]models.Event{}
	for _, r := range a.Records {
		if r.Kind == "events" {
			var e models.Event
			_ = json.Unmarshal(r.Payload, &e)
			if e.EventType == "email" || e.EventType == "email_internal" || e.EventType == "catatan_meeting" {
				if len(e.DealIDs) != 0 {
					t.Fatal("invented interaction deal link")
				}
				for _, account := range e.AccountIDs {
					interactions[account]++
				}
			}
			if e.EventType == "COMMERCIAL_DECISION" {
				decisions[e.EventID] = e
			}
		}
		if r.Kind == "evidence" {
			var e models.Evidence
			_ = json.Unmarshal(r.Payload, &e)
			if e.SourceFile == "interactions.jsonl" && len(e.EventIDs) != 1 {
				t.Fatal("interaction evidence lacks reverse event reference")
			}
			if e.SourceFile == "contact_employment_history.csv" && e.SourceRecordID != "" {
				t.Fatal("invented native row ID")
			}
			if e.SourceFile == "decision_log.csv" && (e.SourceField == "alasan" || e.SourceField == "status_janji") && r.AvailableFrom != repository.MaxAsOf {
				t.Fatal("retrospective field leaked")
			}
		}
	}
	if interactions["P04"] != 3 || interactions["P05"] != 0 {
		t.Fatal("P04/P05 interaction counts", interactions)
	}
	if len(decisions["event:decision:D-2025-06"].DealIDs) != 1 || decisions["event:decision:D-2025-06"].DealIDs[0] != "DL-007" {
		t.Fatal("native decision/deal lost")
	}
	if len(a.SourcePartitions) != 480 {
		t.Fatal("monthly backing containers required")
	}
	backed := 0
	for _, p := range a.SourcePartitions {
		backed += len(p.Rows)
		if len(p.Keys) != len(p.Rows) {
			t.Fatal("missing source key")
		}
	}
	if backed != 226300 {
		t.Fatal("source backing rows lost")
	}
}

func TestFullSourceUsageCutoffMissingAndDistinctOutlets(t *testing.T) {
	rows := []map[string]string{{"tanggal": "2026-09-01", "outlet_id": "O1", "jumlah_transaksi": "10", "transaksi_offline_tersinkron": ""}, {"tanggal": "2026-09-02", "outlet_id": "O1", "jumlah_transaksi": "20", "transaksi_offline_tersinkron": "3"}, {"tanggal": "2026-09-03", "outlet_id": "O2", "jumlah_transaksi": "90", "transaksi_offline_tersinkron": "0"}}
	s := summarizeUsage(rows, "2026-09-02")
	if s.Transactions != 30 || s.ObservedRows != 2 || len(s.OutletIDs) != 1 || s.OfflineMissingCount != 1 || s.OfflineKnownTotal != 3 || s.ObservedThrough != "2026-09-02" {
		t.Fatal("usage cutoff or missingness changed", s)
	}
	if monthEnd("2026-02") != "2026-02-28" {
		t.Fatal("feature month endpoint")
	}
}
