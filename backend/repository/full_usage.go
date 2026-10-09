package repository

import (
	"encoding/json"
	"github.com/nopaalh/Relio/backend/models"
	"sort"
	"strconv"
	"strings"
)

func fullUsageEvidence(raw fullSelection, id string, s models.SnapshotContext) (models.EvidenceResult, error) {
	tuple := strings.Split(strings.TrimPrefix(id, "ev:usage:"), "|")
	if len(tuple) != 2 {
		return models.EvidenceResult{}, &RepositoryError{Code: InvalidQuery}
	}
	if !s.Access.AllowsAccount(tuple[0]) {
		return models.EvidenceResult{}, &RepositoryError{Code: AccessDenied}
	}
	if _, e := models.ParseDate(tuple[1] + "-01"); e != nil {
		return models.EvidenceResult{}, &RepositoryError{Code: InvalidQuery}
	}
	if models.Date(tuple[1]+"-01") > s.AsOf {
		return models.EvidenceResult{}, &RepositoryError{Code: NotVisible}
	}
	for _, p := range raw.SourcePartitions {
		if p.ID != "usage-source:"+strings.Join(tuple, "|") {
			continue
		}
		if p.Hash != FullSourceDigest(p) || raw.Manifest.PartitionRoots["source:"+p.ID] != p.Hash || p.AccountID != tuple[0] {
			return models.EvidenceResult{}, &RepositoryError{Code: DataNotReady}
		}
		sum := struct {
			Transactions        int64    `json:"transactions"`
			OfflineKnownTotal   int64    `json:"offline_known_total"`
			OfflineMissingCount int      `json:"offline_missing_count"`
			ObservedRows        int      `json:"observed_rows"`
			OutletIDs           []string `json:"outlet_ids"`
			ObservedThrough     string   `json:"observed_through"`
			Basis               string   `json:"basis"`
		}{OutletIDs: []string{}, Basis: "source_daily_rows_through_cutoff"}
		outlets := map[string]bool{}
		for _, row := range p.Rows {
			if row["account_id"] != p.AccountID || !strings.HasPrefix(row["tanggal"], tuple[1]+"-") {
				return models.EvidenceResult{}, &RepositoryError{Code: DataNotReady}
			}
			if _, e := models.ParseDate(row["tanggal"]); e != nil {
				return models.EvidenceResult{}, &RepositoryError{Code: DataNotReady}
			}
			if models.Date(row["tanggal"]) > s.AsOf {
				continue
			}
			n, e := strconv.ParseInt(row["jumlah_transaksi"], 10, 64)
			if e != nil || n < 0 {
				return models.EvidenceResult{}, &RepositoryError{Code: DataNotReady}
			}
			sum.Transactions += n
			sum.ObservedRows++
			outlets[row["outlet_id"]] = true
			if row["transaksi_offline_tersinkron"] == "" {
				sum.OfflineMissingCount++
			} else {
				n, e = strconv.ParseInt(row["transaksi_offline_tersinkron"], 10, 64)
				if e != nil || n < 0 {
					return models.EvidenceResult{}, &RepositoryError{Code: DataNotReady}
				}
				sum.OfflineKnownTotal += n
			}
			if row["tanggal"] > sum.ObservedThrough {
				sum.ObservedThrough = row["tanggal"]
			}
		}
		if sum.ObservedRows == 0 {
			return models.EvidenceResult{}, &RepositoryError{Code: NotVisible}
		}
		for id := range outlets {
			sum.OutletIDs = append(sum.OutletIDs, id)
		}
		sort.Strings(sum.OutletIDs)
		b, _ := json.Marshal(sum)
		at := models.Date(sum.ObservedThrough)
		ev := models.Evidence{EvidenceID: id, SourceFile: p.SourceFile, SourceRecordID: "", RecordIDKind: "aggregate", SourceKey: map[string]string{"account_id": p.AccountID, "bulan": tuple[1], "cutoff": string(s.AsOf)}, SourceChecksum: p.SourceChecksum, SourceField: "jumlah_transaksi,transaksi_offline_tersinkron", SourceDate: &at, TemporalBasis: "event", ContentExcerpt: string(b), ScopeKind: "account", VerificationState: "derived", AccountIDs: []string{p.AccountID}, DealIDs: []string{}, EventIDs: []string{}, EdgeIDs: []string{}, OccurrenceIDs: []string{}}
		return models.EvidenceResult{Meta: fullMeta(s), Evidence: ev}, nil
	}
	if raw.Manifest.PartitionRoots["source:usage-source:"+strings.Join(tuple, "|")] != "" {
		return models.EvidenceResult{}, &RepositoryError{Code: DataNotReady}
	}
	return models.EvidenceResult{}, &RepositoryError{Code: NotFound}
}
