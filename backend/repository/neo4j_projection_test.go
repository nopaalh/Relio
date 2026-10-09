package repository

import (
	"encoding/json"
	"errors"
	"github.com/nopaalh/Relio/backend/models"
	"strings"
	"sync"
	"testing"
)

func readyP04(t *testing.T) storedContext {
	t.Helper()
	raw := preparedP04(t)
	raw.Manifest.Ready = true
	return raw
}
func snapshotP04(t *testing.T, raw storedContext, date string) models.SnapshotContext {
	t.Helper()
	s, err := NewSnapshotContext(models.Date(date), raw.Manifest.DatasetVersion, models.AccessScope{AllowedAccountIDs: []string{"P04"}, AllowedDealIDs: []string{"DL-004"}})
	if err != nil {
		t.Fatal(err)
	}
	return s
}
func requireCode(t *testing.T, err error, want ErrorCode) {
	t.Helper()
	var e *RepositoryError
	if !errors.As(err, &e) || e.Code != want {
		t.Fatalf("want %s, got %v", want, err)
	}
}
func TestNeo4jProjectionDates(t *testing.T) {
	raw := readyP04(t)
	for _, tt := range []struct {
		date string
		n    int
	}{{"2026-09-01", 2}, {"2026-09-22", 3}, {"2026-10-01", 3}} {
		t.Run(tt.date, func(t *testing.T) {
			v, err := projectContext(raw, snapshotP04(t, raw, tt.date))
			if err != nil {
				t.Fatal(err)
			}
			if len(v.Events) != tt.n {
				t.Fatal("wrong event cutoff")
			}
			b, _ := json.Marshal(v)
			if tt.n == 2 && (strings.Contains(string(b), "I0335") || strings.Contains(string(b), "Kami tunda")) {
				t.Fatal("future leaked")
			}
			d := v.Deals[0]
			if tt.date != "2026-10-01" {
				if d.Stage.Value != nil || d.OwnerID.Value != nil || d.PotentialACVIDR.Value != nil || d.PlannedOutlets.Value != nil || len(d.Stage.EvidenceIDs) != 0 {
					t.Fatal("snapshot leaked")
				}
			} else {
				if d.Stage.Value == nil || *d.Stage.Value != "Negosiasi" || *d.PotentialACVIDR.Value != 147000000 || *d.PlannedOutlets.Value != 35 {
					t.Fatal("snapshot values lost")
				}
			}
		})
	}
}
func TestNeo4jProjectionScopeAndManifest(t *testing.T) {
	raw := readyP04(t)
	s := snapshotP04(t, raw, "2026-09-01")
	bad := raw
	bad.Manifest.Ready = false
	_, err := projectContext(bad, s)
	requireCode(t, err, DataNotReady)
	bad = raw
	bad.Manifest.Counts = map[string]int{}
	_, err = projectContext(bad, s)
	requireCode(t, err, DataNotReady)
	s.ContextID = "tampered"
	_, err = projectContext(raw, s)
	requireCode(t, err, ContextMismatch)
	for _, a := range []models.AccessScope{{}, {AllowedAnalogAccountIDs: []string{"P04"}}} {
		empty, _ := NewSnapshotContext("2026-09-01", raw.Manifest.DatasetVersion, a)
		_, err = projectContext(raw, empty)
		requireCode(t, err, AccessDenied)
	}
}
func TestNeo4jProjectionReferenceClosure(t *testing.T) {
	raw := readyP04(t)
	end := models.Date("2026-09-30")
	raw.Edges[0].Value.Validity.ValidTo = &end
	v, err := projectContext(raw, snapshotP04(t, raw, "2026-09-01"))
	if err != nil {
		t.Fatal(err)
	}
	for _, e := range v.Edges {
		if e.Validity.ValidTo != nil {
			t.Fatal("future end leaked")
		}
	}
	evs := map[string]bool{}
	ns := map[string]bool{}
	es := map[string]bool{}
	events := map[string]bool{}
	for _, e := range v.Evidence {
		evs[e.EvidenceID] = true
	}
	for _, n := range v.Nodes {
		ns[n.NodeID] = true
	}
	for _, e := range v.Edges {
		es[e.EdgeID] = true
	}
	for _, e := range v.Events {
		events[e.EventID] = true
	}
	for _, e := range v.Edges {
		if !ns[e.Source] || !ns[e.Target] {
			t.Fatal("missing endpoint")
		}
		for _, p := range e.EvidenceIDs {
			if !evs[p] {
				t.Fatal("missing proof")
			}
		}
	}
	for _, e := range v.Events {
		for _, id := range e.EdgeIDs {
			if !es[id] {
				t.Fatal("future/missing edge")
			}
		}
		for _, id := range e.NodeIDs {
			if !ns[id] {
				t.Fatal("missing node")
			}
		}
	}
	for _, e := range v.Evidence {
		for _, id := range e.EventIDs {
			if !events[id] {
				t.Fatal("future event ref")
			}
		}
		for _, id := range e.EdgeIDs {
			if !es[id] {
				t.Fatal("future edge ref")
			}
		}
	}
}
func TestNeo4jProjectionDoesNotMutate(t *testing.T) {
	raw := readyP04(t)
	before, _ := json.Marshal(raw)
	s := snapshotP04(t, raw, "2026-09-01")
	var wg sync.WaitGroup
	for i := 0; i < 8; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			if _, err := projectContext(raw, s); err != nil {
				t.Error(err)
			}
		}()
	}
	wg.Wait()
	after, _ := json.Marshal(raw)
	if string(before) != string(after) {
		t.Fatal("projection mutated input")
	}
}
