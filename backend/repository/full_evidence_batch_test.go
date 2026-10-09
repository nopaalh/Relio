package repository

import (
	"context"
	"encoding/json"
	"errors"
	"testing"

	"github.com/nopaalh/Relio/backend/models"
)

func newBatchEvidenceRepo(t *testing.T) (*Neo4jFullContextRepository, models.SnapshotContext, *int) {
	t.Helper()
	date := models.Date("2026-01-02")
	makeRecord := func(id, account string, sourceDate models.Date) FullRecord {
		payload, err := json.Marshal(models.Evidence{EvidenceID: id, SourceFile: "DATA-UJI.csv", SourceRecordID: id, SourceChecksum: "DATA-UJI", SourceField: "note", ContentExcerpt: "DATA UJI", ScopeKind: "account", VerificationState: "source_backed", AccountIDs: []string{account}, DealIDs: []string{"D1"}, SourceDate: &sourceDate})
		if err != nil {
			t.Fatal(err)
		}
		return FullRecord{Kind: "evidence", ID: id, PartitionID: "A", AccountIDs: []string{account}, DealIDs: []string{"D1"}, AvailableFrom: "2026-01-01", Payload: payload, PayloadHash: fullHash(payload)}
	}
	artifact := FullArtifact{Manifest: FullManifest{DatasetVersion: "full:data-uji", DealAccountIDs: map[string]string{"D1": "A"}}, Records: []FullRecord{
		makeRecord("E1", "A", "2026-01-01"), makeRecord("E2", "A", "2026-01-02"), makeRecord("E-FUTURE", "A", "2026-01-03"), makeRecord("E-OTHER", "B", "2026-01-02"),
	}}
	if err := SealFullArtifact(&artifact); err != nil {
		t.Fatal(err)
	}
	reads := 0
	r := &Neo4jFullContextRepository{read: func(_ context.Context, q fullQuery) (fullSelection, error) {
		reads++
		out := fullSelection{Manifest: artifact.Manifest, Records: []FullRecord{}, Partitions: []FullPartition{}}
		partitions := map[string]bool{}
		for _, record := range artifact.Records {
			hit := q.LookupID != "" && q.LookupID == record.ID
			if !hit && record.AvailableFrom <= q.AsOf && matches(q.Kinds, record.Kind) {
				for _, account := range record.AccountIDs {
					if matches(q.Accounts, account) {
						hit = true
					}
				}
				if len(record.DealIDs) > 0 {
					hit = false
					for _, id := range record.DealIDs {
						if matches(q.Deals, id) {
							hit = true
						}
					}
				}
			}
			if hit {
				out.Records = append(out.Records, record)
				partitions[record.PartitionID] = true
			}
		}
		for _, partition := range artifact.Partitions {
			if partitions[partition.PartitionID] {
				out.Partitions = append(out.Partitions, partition)
			}
		}
		return out, nil
	}}
	snapshot, err := NewSnapshotContext(date, artifact.Manifest.DatasetVersion, models.AccessScope{AllowedAccountIDs: []string{"A"}, AllowedDealIDs: []string{"D1"}})
	if err != nil {
		t.Fatal(err)
	}
	return r, snapshot, &reads
}

func TestReadEvidenceBatchLoadsScopedSelectionOnce(t *testing.T) {
	r, snapshot, reads := newBatchEvidenceRepo(t)
	got, err := r.ReadEvidenceBatch(context.Background(), []string{"E1", "E2"}, snapshot)
	if err != nil {
		t.Fatal(err)
	}
	if *reads != 1 {
		t.Fatalf("database reads = %d, want one shared batch read", *reads)
	}
	if len(got) != 2 || got[0].Evidence.EvidenceID != "E1" || got[1].Evidence.EvidenceID != "E2" {
		t.Fatalf("unexpected batch result: %+v", got)
	}
	for _, result := range got {
		if result.Meta.ContextID != snapshot.ContextID {
			t.Fatal("batch result lost snapshot context")
		}
	}
}

func TestReadEvidenceBatchPreservesSnapshotAndAccessErrors(t *testing.T) {
	for _, test := range []struct {
		id   string
		code ErrorCode
	}{{"E-FUTURE", NotVisible}, {"E-OTHER", AccessDenied}, {"missing", NotFound}} {
		t.Run(test.id, func(t *testing.T) {
			r, snapshot, _ := newBatchEvidenceRepo(t)
			_, err := r.ReadEvidenceBatch(context.Background(), []string{test.id}, snapshot)
			var repoErr *RepositoryError
			if !errors.As(err, &repoErr) || repoErr.Code != test.code {
				t.Fatalf("error = %v, want %s", err, test.code)
			}
		})
	}
}
