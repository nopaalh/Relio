package repository_test

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"os"
	"reflect"
	"sort"
	"strings"
	"testing"

	"github.com/nopaalh/Relio/backend/models"
)

type fixtureBundle struct {
	Kind       string                        `json:"example_kind"`
	Graph      models.GraphResult            `json:"graph"`
	Timeline   models.TimelineResult         `json:"timeline"`
	Evidence   []models.Evidence             `json:"evidence"`
	Actions    models.ActionCandidatesResult `json:"actions"`
	Ambiguous  models.ParticipantRef         `json:"ambiguous_participant"`
	UnknownACV models.Fact[int64]            `json:"unknown_acv"`
	KnownZero  models.Fact[int64]            `json:"known_zero"`
}

func readContractFixture(t *testing.T) fixtureBundle {
	t.Helper()
	bytes, err := os.ReadFile("testdata/context_contract_cases.json")
	if err != nil {
		t.Fatal(err)
	}
	var bundle fixtureBundle
	if err := json.Unmarshal(bytes, &bundle); err != nil {
		t.Fatal(err)
	}
	if bundle.Kind != "DATA UJI — contract illustration, not competition query" {
		t.Fatal("fixture lacks explicit test-only label")
	}
	return bundle
}

// Test-only validation includes nested facts and participant references.
// It does not replace an adapter's access, identity, or temporal checks.
func validateFixtureNestedRefs(bundle fixtureBundle) error {
	nodes, evidence := map[string]bool{}, map[string]bool{}
	for _, n := range bundle.Graph.Nodes {
		nodes[n.NodeID] = true
	}
	for _, e := range bundle.Evidence {
		evidence[e.EvidenceID] = true
	}
	check := func(set map[string]bool, ids []string, path string) error {
		for _, id := range ids {
			if !set[id] {
				return fmt.Errorf("dangling reference %q at %s", id, path)
			}
		}
		return nil
	}
	var walk func(reflect.Value, string) error
	walk = func(v reflect.Value, path string) error {
		if v.Kind() == reflect.Pointer {
			if v.IsNil() {
				return nil
			}
			return walk(v.Elem(), path)
		}
		if v.CanInterface() {
			if p, ok := v.Interface().(models.ParticipantRef); ok {
				if p.NodeID != nil {
					if err := check(nodes, []string{*p.NodeID}, path+".NodeID"); err != nil {
						return err
					}
				}
				if err := check(nodes, p.CandidateNodeIDs, path+".CandidateNodeIDs"); err != nil {
					return err
				}
			}
		}
		switch v.Kind() {
		case reflect.Struct:
			if ids := v.FieldByName("EvidenceIDs"); ids.IsValid() && ids.CanInterface() {
				if refs, ok := ids.Interface().([]string); ok {
					if err := check(evidence, refs, path+".EvidenceIDs"); err != nil {
						return err
					}
				}
			}
			for i := 0; i < v.NumField(); i++ {
				if err := walk(v.Field(i), path+"."+v.Type().Field(i).Name); err != nil {
					return err
				}
			}
		case reflect.Slice:
			for i := 0; i < v.Len(); i++ {
				if err := walk(v.Index(i), fmt.Sprintf("%s[%d]", path, i)); err != nil {
					return err
				}
			}
		}
		return nil
	}
	return walk(reflect.ValueOf(bundle), "fixture")
}

func TestContractFixtureReferencesAndSourceLocators(t *testing.T) {
	bundle := readContractFixture(t)
	if err := validateFixtureNestedRefs(bundle); err != nil {
		t.Fatal(err)
	}
	nodes, edges, events, evidence, occurrences := map[string]bool{}, map[string]bool{}, map[string]bool{}, map[string]bool{}, map[string]bool{}
	add := func(set map[string]bool, id string) {
		t.Helper()
		if !strings.HasPrefix(id, "TEST-") || set[id] {
			t.Fatalf("invalid/duplicate fixture ID %q", id)
		}
		set[id] = true
	}
	for _, n := range bundle.Graph.Nodes {
		add(nodes, n.NodeID)
	}
	for _, e := range bundle.Graph.Edges {
		add(edges, e.EdgeID)
	}
	for _, e := range bundle.Timeline.Events {
		add(events, e.EventID)
	}
	for _, e := range bundle.Evidence {
		add(evidence, e.EvidenceID)
	}
	for _, c := range bundle.Actions.Items {
		for _, p := range c.Precedents {
			add(occurrences, p.Occurrence.OccurrenceID)
		}
	}
	refs := func(set map[string]bool, ids []string) {
		t.Helper()
		for _, id := range ids {
			if !set[id] {
				t.Fatalf("dangling reference %q", id)
			}
		}
	}
	for _, n := range bundle.Graph.Nodes {
		refs(evidence, n.EvidenceIDs)
		refs(events, n.EventIDs)
	}
	for _, e := range bundle.Graph.Edges {
		refs(nodes, []string{e.Source, e.Target})
		refs(events, e.EventIDs)
		refs(evidence, e.EvidenceIDs)
	}
	for _, e := range bundle.Timeline.Events {
		refs(nodes, e.NodeIDs)
		refs(edges, e.EdgeIDs)
		refs(evidence, e.EvidenceIDs)
	}
	sourceBytes, err := os.ReadFile("testdata/contract_sources.json")
	if err != nil {
		t.Fatal(err)
	}
	digest := sha256.Sum256(sourceBytes)
	var records map[string]struct {
		Text string      `json:"text"`
		Date models.Date `json:"date"`
	}
	if err := json.Unmarshal(sourceBytes, &records); err != nil {
		t.Fatal(err)
	}
	for _, e := range bundle.Evidence {
		refs(events, e.EventIDs)
		refs(edges, e.EdgeIDs)
		refs(occurrences, e.OccurrenceIDs)
		record, ok := records[e.SourceRecordID]
		if !ok || e.SourceFile != "contract_sources.json" || e.RecordIDKind != "native" || e.SourceField != "text" || e.ContentExcerpt != record.Text || e.SourceDate == nil || *e.SourceDate != record.Date || e.SourceChecksum != hex.EncodeToString(digest[:]) {
			t.Fatalf("source locator/excerpt/checksum mismatch: %s", e.EvidenceID)
		}
	}
	for _, candidate := range bundle.Actions.Items {
		if len(candidate.Precedents) == 0 || len(candidate.RelevanceReasons) == 0 {
			t.Fatal("candidate lacks supported precedent/relevance")
		}
		for _, reason := range candidate.RelevanceReasons {
			refs(evidence, reason.DealEvidenceIDs)
			refs(evidence, reason.PrecedentEvidenceIDs)
		}
		for _, p := range candidate.Precedents {
			o := p.Occurrence
			if o.ActionID != candidate.Template.ActionID {
				t.Fatal("occurrence belongs to another template")
			}
			refs(events, []string{o.EventID})
			refs(nodes, o.NodeIDs)
			refs(edges, o.EdgeIDs)
			refs(evidence, o.EvidenceIDs)
			if o.Policy.RequestOccurrenceID != nil {
				refs(occurrences, []string{*o.Policy.RequestOccurrenceID})
			}
		}
	}
}

func TestContractFixtureRejectsDanglingNestedReferences(t *testing.T) {
	missing := "TEST-MISSING"
	for _, tt := range []struct {
		name   string
		mutate func(*fixtureBundle)
	}{
		{"fact evidence", func(b *fixtureBundle) { b.Graph.Nodes[0].Label.EvidenceIDs = []string{missing} }},
		{"actor node", func(b *fixtureBundle) { b.Timeline.Events[0].Actors[0].NodeID = &missing }},
		{"ambiguous candidate", func(b *fixtureBundle) { b.Ambiguous.CandidateNodeIDs = []string{missing} }},
		{"participant evidence", func(b *fixtureBundle) { b.Timeline.Events[0].Actors[0].EvidenceIDs = []string{missing} }},
		{"approval evidence", func(b *fixtureBundle) {
			b.Actions.Items[0].Precedents[1].Occurrence.Policy.ApprovalStatus.EvidenceIDs = []string{missing}
		}},
	} {
		t.Run(tt.name, func(t *testing.T) {
			bundle := readContractFixture(t)
			tt.mutate(&bundle)
			if err := validateFixtureNestedRefs(bundle); err == nil {
				t.Fatal("dangling nested reference accepted")
			}
		})
	}
}

func TestContractFixtureUnknownApprovalAndAccountOnlyStayExplicit(t *testing.T) {
	bundle := readContractFixture(t)
	if bundle.Ambiguous.NodeID != nil || bundle.Ambiguous.VerificationState != "ambiguous" || len(bundle.Ambiguous.CandidateNodeIDs) != 2 {
		t.Fatal("ambiguity promoted to verified identity")
	}
	if bundle.UnknownACV.Value != nil || bundle.UnknownACV.State != "snapshot_only" || bundle.KnownZero.Value == nil || *bundle.KnownZero.Value != 0 || bundle.KnownZero.State != "known" {
		t.Fatal("unknown and known zero collapsed")
	}
	for _, event := range bundle.Timeline.Events {
		if event.ScopeKind != "account" || len(event.DealIDs) != 0 || !reflect.DeepEqual(event.AccountIDs, []string{"TEST-ACCOUNT"}) || event.SourceTimestamp != nil {
			t.Fatal("account-only/date-only event gained invented deal/timestamp")
		}
	}
	ordered := append([]models.Event(nil), bundle.Timeline.Events...)
	sort.Slice(ordered, func(i, j int) bool {
		if ordered[i].EventAt == ordered[j].EventAt {
			return ordered[i].EventID < ordered[j].EventID
		}
		return ordered[i].EventAt < ordered[j].EventAt
	})
	if !reflect.DeepEqual(ordered, bundle.Timeline.Events) {
		t.Fatal("timeline fixture not in stable (date,ID) order")
	}
	if len(bundle.Actions.Items) != 1 || bundle.Actions.ReturnedCount != 1 || bundle.Actions.TotalValidCount == nil || *bundle.Actions.TotalValidCount != 1 || bundle.Actions.Bounds.Truncated {
		t.Fatal("candidate count padded or incomplete count mislabeled")
	}
	got := map[string]models.ActionOccurrence{}
	for _, p := range bundle.Actions.Items[0].Precedents {
		got[p.Occurrence.OccurrenceID] = p.Occurrence
	}
	request10, approval10, request14 := got["TEST-OCC-REQ10"], got["TEST-OCC-APP10"], got["TEST-OCC-REQ14"]
	if request10.Status.Value == nil || *request10.Status.Value != "requested" || request10.Policy.RequestedDiscountBPS.Value == nil || *request10.Policy.RequestedDiscountBPS.Value != 1000 {
		t.Fatal("10% request changed status/value")
	}
	if approval10.Status.Value == nil || *approval10.Status.Value != "approved" || approval10.Policy.ApprovedDiscountBPS.Value == nil || *approval10.Policy.ApprovedDiscountBPS.Value != 1000 || approval10.Policy.RequestOccurrenceID == nil || *approval10.Policy.RequestOccurrenceID != "TEST-OCC-REQ10" {
		t.Fatal("approval no longer tied to its 10% request")
	}
	if request14.Status.Value == nil || *request14.Status.Value != "requested" || request14.Policy.RequestedDiscountBPS.Value == nil || *request14.Policy.RequestedDiscountBPS.Value != 1400 || request14.Policy.ApprovalStatus.Value != nil || request14.Policy.ApprovalStatus.State != "unknown" || request14.Policy.ApprovedDiscountBPS.Value != nil || request14.Policy.DecisionID != nil || request14.Policy.AppliedContractID != nil {
		t.Fatal("old approval transferred to new 14% request")
	}
	for _, occurrence := range got {
		if occurrence.Policy.ConsentStatus.Value != nil || occurrence.OutcomeObserved.Value != nil {
			t.Fatal("fixture invented consent/outcome")
		}
	}
}

// Valid fixture outputs initialize every slice/map, including nested facts.
// This is not automatic normalization of arbitrary production responses.
func TestContractFixtureFactsAndRequiredCollections(t *testing.T) {
	bundle := readContractFixture(t)
	var walk func(reflect.Value)
	walk = func(v reflect.Value) {
		if v.Kind() == reflect.Pointer {
			if !v.IsNil() {
				walk(v.Elem())
			}
			return
		}
		if v.CanInterface() {
			if validator, ok := v.Interface().(interface{ Validate() error }); ok {
				if err := validator.Validate(); err != nil {
					t.Fatalf("invalid fact: %v", err)
				}
			}
		}
		switch v.Kind() {
		case reflect.Struct:
			for i := 0; i < v.NumField(); i++ {
				walk(v.Field(i))
			}
		case reflect.Slice:
			if v.IsNil() {
				t.Fatalf("required collection is nil: %s", v.Type())
			}
			for i := 0; i < v.Len(); i++ {
				walk(v.Index(i))
			}
		case reflect.Map:
			if v.IsNil() {
				t.Fatal("required source key map is nil")
			}
		}
	}
	walk(reflect.ValueOf(bundle))
}
