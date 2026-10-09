package repository_test

import (
	"encoding/json"
	"errors"
	"github.com/nopaalh/Relio/backend/models"
	"github.com/nopaalh/Relio/backend/repository"
	"reflect"
	"testing"
)

func TestSnapshotRejectsInvalidAndFuture(t *testing.T) {
	for _, raw := range []models.Date{"", "2026-02-30", "2026-10-02", "2026-10-01T00:00:00Z", " 2026-10-01"} {
		_, err := repository.NewSnapshotContext(raw, "TEST-DATASET", models.AccessScope{})
		var typed *repository.RepositoryError
		if !errors.As(err, &typed) || typed.Code != repository.InvalidSnapshot {
			t.Fatalf("accepted/incorrect error for %q: %v", raw, err)
		}
	}
	_, err := repository.NewSnapshotContext("2026-10-01", "", models.AccessScope{})
	var typed *repository.RepositoryError
	if !errors.As(err, &typed) || typed.Code != repository.DataNotReady {
		t.Fatalf("missing dataset version = %v", err)
	}
	for _, scope := range []models.AccessScope{
		{AllowedDealIDs: []string{""}}, {AllowedAccountIDs: []string{" "}}, {AllowedAnalogAccountIDs: []string{" TEST-B"}},
	} {
		if _, err := repository.NewSnapshotContext("2026-10-01", "TEST-DATASET", scope); err == nil {
			t.Fatal("blank/malformed allowlist member accepted")
		}
	}
}

func TestContextIDStableAcrossScopeOrder(t *testing.T) {
	scope := models.AccessScope{AllowedAccountIDs: []string{"TEST-B", "TEST-A", "TEST-A"}, AllowedDealIDs: []string{"TEST-D"}, AllowedAnalogAccountIDs: []string{"TEST-Z"}}
	before := append([]string(nil), scope.AllowedAccountIDs...)
	a, err := repository.NewSnapshotContext("2026-10-01", "TEST-DATASET", scope)
	if err != nil {
		t.Fatal(err)
	}
	b, err := repository.NewSnapshotContext("2026-10-01", "TEST-DATASET", models.AccessScope{AllowedAccountIDs: []string{"TEST-A", "TEST-B"}, AllowedDealIDs: []string{"TEST-D"}, AllowedAnalogAccountIDs: []string{"TEST-Z"}})
	if err != nil {
		t.Fatal(err)
	}
	if a.ContextID == "" || a.ContextID != b.ContextID {
		t.Fatal("context hash depends on set order/duplicates")
	}
	if !reflect.DeepEqual(scope.AllowedAccountIDs, before) {
		t.Fatal("constructor mutated server scope")
	}
	scope.AllowedAccountIDs[0] = "TEST-CHANGED"
	if a.Access.AllowedAccountIDs[0] != "TEST-A" {
		t.Fatal("snapshot retained input slice alias")
	}
	if err := repository.ValidateSnapshot(a); err != nil {
		t.Fatal(err)
	}
}

func TestContextIDChangesWithSnapshotAndPermission(t *testing.T) {
	base, err := repository.NewSnapshotContext("2026-10-01", "TEST-DATASET", models.AccessScope{AllowedAccountIDs: []string{"TEST-A"}})
	if err != nil {
		t.Fatal(err)
	}
	for _, tt := range []struct {
		date    models.Date
		version string
		scope   models.AccessScope
	}{
		{"2026-09-01", "TEST-DATASET", base.Access},
		{"2026-10-01", "TEST-DATASET-2", base.Access},
		{"2026-10-01", "TEST-DATASET", models.AccessScope{AllowedAccountIDs: []string{"TEST-B"}}},
		{"2026-10-01", "TEST-DATASET", models.AccessScope{AllowedAccountIDs: []string{"TEST-A"}, AllowedDealIDs: []string{"TEST-D"}}},
		{"2026-10-01", "TEST-DATASET", models.AccessScope{AllowedAccountIDs: []string{"TEST-A"}, AllowedAnalogAccountIDs: []string{"TEST-B"}}},
		{"2026-10-01", "TEST-DATASET", models.AccessScope{AllowedAccountIDs: []string{"TEST-A"}, AllowCompanyEvidence: true}},
	} {
		got, err := repository.NewSnapshotContext(tt.date, tt.version, tt.scope)
		if err != nil {
			t.Fatal(err)
		}
		if got.ContextID == base.ContextID {
			t.Fatal("context reused across snapshot/version/permission")
		}
	}
	// Delimiters inside IDs cannot collide with multiple IDs.
	a, _ := repository.NewSnapshotContext("2026-10-01", "TEST-DATASET", models.AccessScope{AllowedAccountIDs: []string{"A,B"}})
	b, _ := repository.NewSnapshotContext("2026-10-01", "TEST-DATASET", models.AccessScope{AllowedAccountIDs: []string{"A", "B"}})
	if a.ContextID == b.ContextID {
		t.Fatal("ambiguous scope encoding")
	}
}

func TestValidateSnapshotRejectsTampering(t *testing.T) {
	base, err := repository.NewSnapshotContext("2026-10-01", "TEST-DATASET", models.AccessScope{})
	if err != nil {
		t.Fatal(err)
	}
	for _, mutate := range []func(*models.SnapshotContext){
		func(s *models.SnapshotContext) { s.AsOf = "2026-09-01" },
		func(s *models.SnapshotContext) { s.DatasetVersion = "TEST-OTHER" },
		func(s *models.SnapshotContext) { s.ContractVersion = "TEST-final-not-agreed" },
		func(s *models.SnapshotContext) { s.CalendarZone = "UTC" },
		func(s *models.SnapshotContext) { s.ContextID = "" },
		func(s *models.SnapshotContext) { s.Access.AllowCompanyEvidence = true },
	} {
		changed := base
		mutate(&changed)
		var typed *repository.RepositoryError
		if err := repository.ValidateSnapshot(changed); !errors.As(err, &typed) || typed.Code != repository.ContextMismatch {
			t.Fatalf("tampered context accepted: %v", err)
		}
	}
}

func TestEmptyScopeAndAnalogPermissionRemainSeparate(t *testing.T) {
	empty, err := repository.NewSnapshotContext("2026-10-01", "TEST-DATASET", models.AccessScope{})
	if err != nil {
		t.Fatal(err)
	}
	if empty.Access.AllowsAccount("TEST-A") || empty.Access.AllowsDeal("TEST-D", "TEST-A") || empty.Access.AllowsAnalogAccount("TEST-A") || empty.Access.AllowCompanyEvidence {
		t.Fatal("empty scope granted access")
	}
	analog := models.AccessScope{AllowedAnalogAccountIDs: []string{"TEST-A"}}
	if !analog.AllowsAnalogAccount("TEST-A") || analog.AllowsAccount("TEST-A") || analog.AllowsDeal("TEST-D", "TEST-A") {
		t.Fatal("analog permission became primary access")
	}
	primary := models.AccessScope{AllowedAccountIDs: []string{"TEST-A"}, AllowedDealIDs: []string{"TEST-D"}}
	if !primary.AllowsDeal("TEST-D", "TEST-A") || primary.AllowsDeal("TEST-D", "TEST-B") || primary.AllowsDeal("TEST-OTHER", "TEST-A") || primary.AllowsAnalogAccount("TEST-A") {
		t.Fatal("scope membership not enforced")
	}
	if (models.AccessScope{AllowedAccountIDs: []string{""}, AllowedDealIDs: []string{""}}).AllowsDeal("", "") {
		t.Fatal("empty IDs granted access")
	}
	// Serialization of a correlation context must not disclose the server allowlist.
	bytes, err := json.Marshal(empty)
	if err != nil {
		t.Fatal(err)
	}
	var wire map[string]any
	if err := json.Unmarshal(bytes, &wire); err != nil {
		t.Fatal(err)
	}
	if _, ok := wire["Access"]; ok {
		t.Fatal("server scope serialized")
	}
	if _, ok := wire["access"]; ok {
		t.Fatal("server scope serialized")
	}
}
