package repository

import (
	"context"
	"encoding/json"
	"github.com/nopaalh/Relio/backend/models"
	"os"
	"strings"
	"testing"
	"time"
)

type fullRepositories interface {
	DealFactsRepository
	GraphRepository
	TimelineRepository
	EvidenceRepository
	ActionRepository
}

func fullDemoAcceptance(t *testing.T, r fullRepositories, version string, allProofs bool) {
	t.Helper()
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Minute)
	defer cancel()
	for _, c := range []struct{ deal, account string }{{"DL-001", "P01"}, {"DL-002", "P02"}, {"DL-003", "P03"}, {"DL-004", "P04"}, {"DL-005", "P05"}} {
		for _, at := range []models.Date{"2026-09-01", "2026-09-22", MaxAsOf} {
			s, e := NewSnapshotContext(at, version, models.AccessScope{AllowedAccountIDs: []string{c.account}, AllowedDealIDs: []string{c.deal}, AllowedAnalogAccountIDs: []string{"C01", "C23", "C24"}, AllowCompanyEvidence: true})
			if e != nil {
				t.Fatal(e)
			}
			started := time.Now()
			d, e := r.FindFacts(ctx, c.deal, s)
			if (c.account == "P03" && at < "2026-09-15") || (c.account == "P05" && at < "2026-09-26") {
				requireFullCode(t, e, NotVisible)
				t.Logf("%s %s correctly not visible before creation", c.deal, at)
				continue
			}
			if e != nil {
				t.Fatalf("%s %s deal: %v", c.deal, at, e)
			}
			if d.Deal.AccountID != c.account {
				t.Fatal("native account mismatch")
			}
			if at < MaxAsOf && d.Deal.Stage.Value != nil {
				t.Fatal("historical stage invented")
			}
			tl, e := r.ReadTimeline(ctx, c.deal, s, models.TimelineOptions{})
			if e != nil {
				t.Fatal(e)
			}
			interactions := 0
			for _, event := range tl.Events {
				if event.EventAt > at {
					t.Fatal("future event")
				}
				if strings.HasPrefix(event.EventID, "event:interaction:") {
					interactions++
				}
				for proofIndex, id := range event.EvidenceIDs {
					if !allProofs && proofIndex > 0 {
						break
					}
					ev, e := r.ReadEvidence(ctx, id, s)
					if e != nil {
						t.Fatalf("timeline proof %s: %v", id, e)
					}
					if ev.Evidence.SourceDate != nil && *ev.Evidence.SourceDate > at {
						t.Fatal("future proof")
					}
				}
			}
			if c.account == "P04" {
				want := 3
				if at == "2026-09-01" {
					want = 2
					futureFocus := "event:interaction:I0335"
					_, err := r.ReadGraph(ctx, c.deal, s, models.GraphOptions{FocusEventID: &futureFocus})
					requireFullCode(t, err, NotVisible)
				}
				if interactions != want {
					t.Fatalf("P04 interactions got %d want %d", interactions, want)
				}
			}
			if c.account == "P05" && interactions != 0 {
				t.Fatal("invented P05 interactions")
			}
			g, e := r.ReadGraph(ctx, c.deal, s, models.GraphOptions{Depth: 2})
			if e != nil {
				t.Fatal(e)
			}
			nodes := map[string]bool{}
			for _, n := range g.Nodes {
				nodes[n.NodeID] = true
			}
			for _, edge := range g.Edges {
				if !nodes[edge.Source] || !nodes[edge.Target] {
					t.Fatal("graph endpoint absent")
				}
			}
			candidates, e := r.ReadActionCandidates(ctx, c.deal, s, models.CandidateOptions{RelevanceRuleVersion: FullRelevanceVersion})
			if e != nil {
				t.Fatal(e)
			}
			if candidates.Meta.ContextID != tl.Meta.ContextID || g.Meta.ContextID != d.Meta.ContextID {
				t.Fatal("context differs")
			}
			for _, candidate := range candidates.Items {
				for _, p := range candidate.Precedents {
					if p.Occurrence.EventAt > at || len(p.Occurrence.EvidenceIDs) == 0 {
						t.Fatal("invalid precedent")
					}
					if p.Relation == "analog_precedent" && p.Occurrence.Policy.ConsentStatus.Value != nil {
						t.Fatal("inherited consent")
					}
				}
			}
			ids := []string{}
			for _, candidate := range candidates.Items {
				if len(ids) < 4 {
					ids = append(ids, candidate.Template.ActionID)
				}
			}
			bytes := 0
			if len(ids) > 0 {
				bundle, e := r.ReadSelectedActions(ctx, c.deal, s, ids, FullRelevanceVersion)
				if e != nil {
					t.Fatal(e)
				}
				b, _ := json.Marshal(bundle)
				bytes = len(b)
			}
			t.Logf("%s %s nodes=%d edges=%d timeline=%d candidates=%d selected=%d bundle_bytes=%d sequence_ms=%d", c.deal, at, len(g.Nodes), len(g.Edges), len(tl.Events), len(candidates.Items), len(ids), bytes, time.Since(started).Milliseconds())
		}
	}
	s, _ := NewSnapshotContext("2026-09-22", version, models.AccessScope{AllowedAccountIDs: []string{"C01"}, AllowedDealIDs: []string{"DL-008"}})
	ev, e := r.ReadEvidence(ctx, "ev:usage:C01|2026-09", s)
	if e != nil {
		t.Fatalf("partial usage: %v", e)
	}
	if ev.Evidence.SourceDate == nil || *ev.Evidence.SourceDate > s.AsOf {
		t.Fatal("usage future leak")
	}
	_, e = r.ReadEvidence(ctx, "ev:interactions.jsonl:I0335:isi", s)
	requireFullCode(t, e, AccessDenied)
}

// Prepared acceptance substitutes only transport; all projection/query logic remains real.
func TestFullPreparedAcceptance(t *testing.T) {
	path := os.Getenv("RELIO_FULL_ARTIFACT_TEST")
	if path == "" {
		t.Skip("explicit local generated artifact opt-in")
	}
	b, e := os.ReadFile(path)
	if e != nil {
		t.Fatal(e)
	}
	var a FullArtifact
	if e = json.Unmarshal(b, &a); e != nil {
		t.Fatal(e)
	}
	if e = ValidateFullArtifact(a); e != nil {
		t.Fatal(e)
	}
	r := &Neo4jFullContextRepository{read: func(_ context.Context, q fullQuery) (fullSelection, error) {
		out := fullSelection{Manifest: a.Manifest, Records: []FullRecord{}, Partitions: []FullPartition{}, SourcePartitions: []FullSourcePartition{}}
		parts := map[string]bool{}
		for _, r := range a.Records {
			lookupKind := q.LookupKind
			if lookupKind == "" {
				lookupKind = "evidence"
			}
			hit := r.Kind == lookupKind && q.LookupID != "" && r.ID == q.LookupID
			if !hit && r.AvailableFrom <= q.AsOf && matches(q.Kinds, r.Kind) {
				if len(r.AccountIDs) == 0 {
					hit = q.Company
				} else {
					for _, acc := range r.AccountIDs {
						if matches(q.Accounts, acc) {
							hit = true
							break
						}
					}
				}
				if hit && len(r.DealIDs) > 0 {
					hit = false
					for _, d := range r.DealIDs {
						if matches(q.Deals, d) {
							hit = true
						}
					}
					for _, acc := range r.AccountIDs {
						if len(q.Analogs) > 0 && matches(q.Analogs, acc) {
							hit = true
						}
					}
				}
			}
			if hit {
				out.Records = append(out.Records, r)
				parts[r.PartitionID] = true
			}
		}
		for _, p := range a.Partitions {
			if parts[p.PartitionID] {
				out.Partitions = append(out.Partitions, p)
			}
		}
		for _, p := range a.SourcePartitions {
			if "ev:usage:"+strings.TrimPrefix(p.ID, "usage-source:") == q.LookupID {
				out.SourcePartitions = append(out.SourcePartitions, p)
			}
		}
		return out, nil
	}}
	ctx := context.Background()
	scope := models.AccessScope{AllowedAccountIDs: a.Manifest.AccountIDs, AllowedDealIDs: a.Manifest.DealIDs, AllowCompanyEvidence: true}
	s, _ := NewSnapshotContext(MaxAsOf, a.Manifest.DatasetVersion, scope)
	list, e := r.ListFacts(ctx, s, models.DealListOptions{})
	if e != nil || len(list.Items) != 22 {
		t.Fatalf("all22 deals: %d %v", len(list.Items), e)
	}
	for _, id := range a.Manifest.DealIDs {
		if _, e = r.FindFacts(ctx, id, s); e != nil {
			t.Fatal(id, e)
		}
	}
	fullDemoAcceptance(t, r, a.Manifest.DatasetVersion, true)
}
func TestNeo4jFullLive(t *testing.T) {
	if os.Getenv("RELIO_FULL_INTEGRATION") != "1" {
		t.Skip("explicit read-only full Aura opt-in")
	}
	for _, key := range []string{"NEO4J_URI", "NEO4J_USERNAME", "NEO4J_PASSWORD", "NEO4J_DATABASE", "RELIO_FULL_DATASET_VERSION"} {
		if os.Getenv(key) == "" {
			t.Fatalf("required key absent: %s", key)
		}
	}
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	r, e := NewNeo4jFullContextRepository(ctx, Neo4jConfig{URI: os.Getenv("NEO4J_URI"), Username: os.Getenv("NEO4J_USERNAME"), Password: os.Getenv("NEO4J_PASSWORD"), Database: os.Getenv("NEO4J_DATABASE")})
	if e != nil {
		t.Fatal(e)
	}
	defer r.Close(context.Background())
	privacySnapshot, _ := NewSnapshotContext(MaxAsOf, os.Getenv("RELIO_FULL_DATASET_VERSION"), models.AccessScope{AllowedAccountIDs: []string{"P04"}, AllowedDealIDs: []string{"DL-004"}, AllowCompanyEvidence: true})
	privacyRaw, err := r.read(ctx, fullQueryFor(privacySnapshot, "context"))
	if err != nil {
		t.Fatal(err)
	}
	checkedUnscoped := false
	for _, record := range privacyRaw.Records {
		if record.Kind != "evidence" || len(record.AccountIDs) != 0 {
			continue
		}
		var evidence models.Evidence
		if json.Unmarshal(record.Payload, &evidence) != nil {
			t.Fatal("invalid evidence")
		}
		if evidence.SourceFile == "contact_employment_history.csv" {
			_, err = r.ReadEvidence(ctx, evidence.EvidenceID, privacySnapshot)
			requireFullCode(t, err, AccessDenied)
			checkedUnscoped = true
			break
		}
	}
	if !checkedUnscoped {
		t.Fatal("expected stored unscoped employment coverage")
	}
	fullDemoAcceptance(t, r, os.Getenv("RELIO_FULL_DATASET_VERSION"), false)
}
