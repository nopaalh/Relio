package services

import (
	"context"
	"fmt"
	"sync/atomic"
	"testing"
	"time"

	"github.com/nopaalh/Relio/backend/models"
)

func TestReadAssessmentEvidenceRunsBoundedReadsConcurrently(t *testing.T) {
	var active, maximum atomic.Int32
	ids := []string{"E1", "E2", "E3", "E4", "E5", "E6", "E7", "E8", "E9"}
	got, err := readAssessmentEvidence(context.Background(), ids, func(_ context.Context, id string) (models.EvidenceResult, error) {
		current := active.Add(1)
		defer active.Add(-1)
		for previous := maximum.Load(); current > previous; previous = maximum.Load() {
			if maximum.CompareAndSwap(previous, current) {
				break
			}
		}
		time.Sleep(10 * time.Millisecond)
		return models.EvidenceResult{Evidence: models.Evidence{EvidenceID: id}}, nil
	})
	if err != nil {
		t.Fatal(err)
	}
	if maximum.Load() < 2 || maximum.Load() > 8 {
		t.Fatalf("max concurrent reads = %d, want 2..8", maximum.Load())
	}
	if len(got) != len(ids) {
		t.Fatalf("got %d results, want %d", len(got), len(ids))
	}
	for i, id := range ids {
		if got[i].Evidence.EvidenceID != id {
			t.Fatalf("result[%d] = %q, want %q", i, got[i].Evidence.EvidenceID, id)
		}
	}
}

func TestReadAssessmentEvidencePreservesReadError(t *testing.T) {
	want := fmt.Errorf("DATA UJI read failed")
	_, err := readAssessmentEvidence(context.Background(), []string{"E1"}, func(context.Context, string) (models.EvidenceResult, error) { return models.EvidenceResult{}, want })
	if err != want {
		t.Fatalf("error = %v, want original read error", err)
	}
}
