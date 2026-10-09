package repository

import (
	"context"
	"encoding/json"
	"github.com/nopaalh/Relio/backend/models"
	"testing"
)

func TestFullUsagePartialMonthNoFutureRows(t *testing.T) {
	a, r := fullFixture(t)
	p := FullSourcePartition{ID: "usage-source:A1|2026-09", SourceFile: "product_usage_daily.csv", SourceChecksum: "fixture", AccountID: "A1", Date: "2026-09-01", Rows: []map[string]string{{"account_id": "A1", "tanggal": "2026-09-01", "outlet_id": "O1", "jumlah_transaksi": "7", "transaksi_offline_tersinkron": ""}, {"account_id": "A1", "tanggal": "2026-09-02", "outlet_id": "O1", "jumlah_transaksi": "4", "transaksi_offline_tersinkron": "2"}, {"account_id": "A1", "tanggal": "2026-09-30", "outlet_id": "O2", "jumlah_transaksi": "100", "transaksi_offline_tersinkron": "9"}}, Keys: []map[string]string{{"tanggal": "2026-09-01", "outlet_id": "O1"}, {"tanggal": "2026-09-02", "outlet_id": "O1"}, {"tanggal": "2026-09-30", "outlet_id": "O2"}}}
	a.SourcePartitions = []FullSourcePartition{p}
	if e := SealFullArtifact(&a); e != nil {
		t.Fatal(e)
	}
	r.read = func(context.Context, fullQuery) (fullSelection, error) {
		return fullSelection{Manifest: a.Manifest, Partitions: a.Partitions, Records: a.Records, SourcePartitions: a.SourcePartitions}, nil
	}
	got, e := r.ReadEvidence(context.Background(), "ev:usage:A1|2026-09", fullSnapshot(t, "2026-09-22"))
	if e != nil {
		t.Fatal(e)
	}
	var v struct {
		Transactions        int
		OfflineMissingCount int      `json:"offline_missing_count"`
		OutletIDs           []string `json:"outlet_ids"`
	}
	if e = json.Unmarshal([]byte(got.Evidence.ContentExcerpt), &v); e != nil {
		t.Fatal(e)
	}
	if v.Transactions != 11 || v.OfflineMissingCount != 1 || len(v.OutletIDs) != 1 || got.Evidence.SourceDate == nil || *got.Evidence.SourceDate != models.Date("2026-09-02") {
		t.Fatal("future/missing/distinct cutoff incorrect", got.Evidence)
	}
}
