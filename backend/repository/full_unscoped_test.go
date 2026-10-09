package repository

import (
	"context"
	"encoding/json"
	"github.com/nopaalh/Relio/backend/models"
	"testing"
)

func TestFullUnscopedExternalEmploymentIsNotCompanyPermission(t *testing.T) {
	a, r := fullFixture(t)
	for i := range a.Records {
		if a.Records[i].ID != "foreign" {
			continue
		}
		var ev models.Evidence
		_ = json.Unmarshal(a.Records[i].Payload, &ev)
		ev.SourceFile = "contact_employment_history.csv"
		ev.ScopeKind = "company"
		ev.AccountIDs = []string{}
		ev.SourceKey = map[string]string{"contact_id": "FOREIGN_ILLUSTRATION", "account_id": "", "organisasi": "External_Illustration"}
		a.Records[i].AccountIDs = []string{}
		a.Records[i].Payload, _ = json.Marshal(ev)
	}
	if e := SealFullArtifact(&a); e != nil {
		t.Fatal(e)
	}
	r.read = func(context.Context, fullQuery) (fullSelection, error) {
		return fullSelection{Manifest: a.Manifest, Partitions: a.Partitions, Records: a.Records}, nil
	}
	s, _ := NewSnapshotContext(MaxAsOf, a.Manifest.DatasetVersion, models.AccessScope{AllowedAccountIDs: []string{"A1"}, AllowedDealIDs: []string{"D1"}, AllowCompanyEvidence: true})
	_, e := r.ReadEvidence(context.Background(), "foreign", s)
	requireFullCode(t, e, AccessDenied)
}
