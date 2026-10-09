package main

import (
	"encoding/json"
	"fmt"
	"github.com/nopaalh/Relio/backend/models"
	"github.com/nopaalh/Relio/backend/repository"
	"sort"
	"strconv"
	"time"
)

type usageSummary struct {
	Transactions        int64    `json:"transactions"`
	OfflineKnownTotal   int64    `json:"offline_known_total"`
	OfflineMissingCount int      `json:"offline_missing_count"`
	ObservedRows        int      `json:"observed_rows"`
	OutletIDs           []string `json:"outlet_ids"`
	ObservedThrough     string   `json:"observed_through"`
	Basis               string   `json:"basis"`
}

func summarizeUsage(rows []map[string]string, cutoff string) usageSummary {
	out := usageSummary{OutletIDs: []string{}, Basis: "source_daily_rows_through_cutoff"}
	outlets := map[string]bool{}
	for _, x := range rows {
		if x["tanggal"] > cutoff {
			continue
		}
		n, _ := strconv.ParseInt(x["jumlah_transaksi"], 10, 64)
		out.Transactions += n
		out.ObservedRows++
		outlets[x["outlet_id"]] = true
		if x["transaksi_offline_tersinkron"] == "" {
			out.OfflineMissingCount++
		} else {
			n, _ := strconv.ParseInt(x["transaksi_offline_tersinkron"], 10, 64)
			out.OfflineKnownTotal += n
		}
		if x["tanggal"] > out.ObservedThrough {
			out.ObservedThrough = x["tanggal"]
		}
	}
	for id := range outlets {
		out.OutletIDs = append(out.OutletIDs, id)
	}
	sort.Strings(out.OutletIDs)
	return out
}
func monthEnd(month string) models.Date {
	t, _ := time.Parse("2006-01", month)
	return date(t.AddDate(0, 1, 0).AddDate(0, 0, -1).Format(time.DateOnly))
}
func (b *builder) usage() {
	groups := map[string][]map[string]string{}
	monthly := map[string][]map[string]string{}
	for _, row := range b.src.rows["product_usage_daily.csv"] {
		key := row["account_id"] + "|" + row["tanggal"][:7]
		groups[key] = append(groups[key], row)
		monthly[row["account_id"]+"|"+row["tanggal"][:7]] = append(monthly[row["account_id"]+"|"+row["tanggal"][:7]], row)
	}
	keys := []string{}
	for k := range groups {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	for _, k := range keys {
		rows := groups[k]
		a, at := rows[0]["account_id"], date(rows[0]["tanggal"][:7]+"-01")
		keys := []map[string]string{}
		for _, row := range rows {
			keys = append(keys, map[string]string{"tanggal": row["tanggal"], "outlet_id": row["outlet_id"]})
		}
		b.sourceParts = append(b.sourceParts, repository.FullSourcePartition{ID: "usage-source:" + k, SourceFile: "product_usage_daily.csv", SourceChecksum: b.src.hashes["product_usage_daily.csv"], AccountID: a, Date: at, Rows: rows, Keys: keys})
	}
	keys = []string{}
	for k := range monthly {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	for _, k := range keys {
		rows := monthly[k]
		a, month := rows[0]["account_id"], rows[0]["tanggal"][:7]
		at := monthEnd(month)
		summary := summarizeUsage(rows, string(at))
		blob, _ := json.Marshal(summary)
		eid := "ev:usage:" + k
		e := models.Evidence{EvidenceID: eid, SourceFile: "product_usage_daily.csv", RecordIDKind: "aggregate", SourceKey: map[string]string{"account_id": a, "bulan": month}, SourceChecksum: b.src.hashes["product_usage_daily.csv"], SourceDate: ptr(at), SourceField: "jumlah_transaksi,transaksi_offline_tersinkron", TemporalBasis: "event", ContentExcerpt: string(blob), ScopeKind: "account", VerificationState: "derived", AccountIDs: []string{a}, DealIDs: []string{}, EventIDs: []string{}, EdgeIDs: []string{}, OccurrenceIDs: []string{}}
		b.put("evidence", eid, a, "", at, e)
		nid := "usage:" + k
		b.node(nid, "usage_aggregate", k, a, at, known(fmt.Sprintf("%s: %d transaksi (observed rows)", month, summary.Transactions), "event", eid), []string{eid})
		b.node("account:"+a, "account", a, a, at, known(a, "event", eid), []string{eid})
		b.edge("HAS_USAGE", "account:"+a, nid, a, "", at, []string{eid}, []string{}, models.Validity{TemporalBasis: "event"})
	}
	for _, row := range b.src.rows["feature_usage_monthly.csv"] {
		a, at := row["account_id"], monthEnd(row["bulan"])
		key := map[string]string{"bulan": row["bulan"], "account_id": a, "feature_id": row["feature_id"]}
		ev := b.evidence("feature_usage_monthly.csv", row, key, "", "pengguna_aktif", a, "", at, "event")
		id := "feature-usage:" + repository.FullDigestJSON(key)
		b.node(id, "feature_usage", id, a, at, known(row["bulan"]+": "+row["feature_id"]+" pengguna_aktif="+row["pengguna_aktif"], "event", ev.EvidenceID), []string{ev.EvidenceID})
		b.edge("HAS_FEATURE_USAGE", "account:"+a, id, a, "", at, []string{ev.EvidenceID}, []string{}, models.Validity{TemporalBasis: "event"})
		b.edge("MEASURES_FEATURE", id, "feature:"+row["feature_id"], a, "", at, []string{ev.EvidenceID}, []string{}, models.Validity{TemporalBasis: "event"})
	}
}
