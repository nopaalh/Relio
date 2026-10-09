package repository

import (
	"context"
	"encoding/json"
	"github.com/nopaalh/Relio/backend/models"
	"testing"
)

func TestFullFutureFocusIsNotVisibleNotMissing(t *testing.T) {
	a, r := fullFixture(t)
	p, _ := json.Marshal(models.Event{EventID: "future-event", EventAt: MaxAsOf, ScopeKind: "account", AccountIDs: []string{"A1"}})
	a.Records = append(a.Records, FullRecord{Kind: "events", ID: "future-event", PartitionID: "A1|future-events", AccountIDs: []string{"A1"}, DealIDs: []string{}, AvailableFrom: MaxAsOf, Payload: p})
	if e := SealFullArtifact(&a); e != nil {
		t.Fatal(e)
	}
	r.read = func(_ context.Context, q fullQuery) (fullSelection, error) {
		records := []FullRecord{}
		for _, x := range a.Records {
			if x.AvailableFrom <= q.AsOf || x.ID == q.LookupID {
				records = append(records, x)
			}
		}
		return fullSelection{Manifest: a.Manifest, Partitions: a.Partitions, Records: records}, nil
	}
	id := "future-event"
	_, e := r.ReadGraph(context.Background(), "D1", fullSnapshot(t, "2026-09-22"), models.GraphOptions{FocusEventID: &id})
	requireFullCode(t, e, NotVisible)
}
