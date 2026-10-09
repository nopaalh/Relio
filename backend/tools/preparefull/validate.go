package main

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"github.com/nopaalh/Relio/backend/models"
	"strconv"
	"strings"
	"time"
)

func hashBytes(b []byte) string { s := sha256.Sum256(b); return hex.EncodeToString(s[:]) }
func validateSources(s sources) error {
	sets := map[string]map[string]bool{}
	native := map[string]string{"crm_accounts.csv": "account_id", "crm_contacts.csv": "contact_id", "crm_deals.csv": "deal_id", "employees.csv": "employee_id", "interactions.jsonl": "interaction_id", "outlets.csv": "outlet_id", "contracts_billing.csv": "contract_id", "decision_log.csv": "decision_id", "support_tickets.csv": "ticket_id", "bugs.csv": "bug_id", "releases.csv": "versi", "features.csv": "feature_id"}
	for file, key := range native {
		sets[file] = map[string]bool{}
		for _, row := range s.rows[file] {
			id := row[key]
			if id == "" || sets[file][id] {
				return fmt.Errorf("%s: empty/duplicate native ID", file)
			}
			sets[file][id] = true
		}
	}
	type fk struct{ file, col, target string }
	joins := []fk{{"crm_accounts.csv", "account_owner_id", "employees.csv"}, {"crm_accounts.csv", "champion_contact_id", "crm_contacts.csv"}, {"crm_contacts.csv", "account_id_saat_ini", "crm_accounts.csv"}, {"contact_employment_history.csv", "contact_id", "crm_contacts.csv"}, {"contact_employment_history.csv", "account_id", "crm_accounts.csv"}, {"crm_deals.csv", "account_id", "crm_accounts.csv"}, {"crm_deals.csv", "owner_id", "employees.csv"}, {"outlets.csv", "account_id", "crm_accounts.csv"}, {"product_usage_daily.csv", "outlet_id", "outlets.csv"}, {"product_usage_daily.csv", "account_id", "crm_accounts.csv"}, {"product_usage_daily.csv", "versi_aplikasi", "releases.csv"}, {"feature_usage_monthly.csv", "account_id", "crm_accounts.csv"}, {"feature_usage_monthly.csv", "feature_id", "features.csv"}, {"contracts_billing.csv", "account_id", "crm_accounts.csv"}, {"contracts_billing.csv", "decision_id", "decision_log.csv"}, {"decision_log.csv", "account_id", "crm_accounts.csv"}, {"decision_log.csv", "deal_id", "crm_deals.csv"}, {"decision_log.csv", "diminta_oleh", "employees.csv"}, {"decision_log.csv", "diputuskan_oleh", "employees.csv"}, {"decision_log.csv", "bukti_interaction_id", "interactions.jsonl"}, {"decision_log.csv", "fitur_dijanjikan", "features.csv"}, {"interactions.jsonl", "account_id", "crm_accounts.csv"}, {"interactions.jsonl", "membalas_id", "interactions.jsonl"}, {"support_tickets.csv", "account_id", "crm_accounts.csv"}, {"support_tickets.csv", "outlet_id", "outlets.csv"}, {"support_tickets.csv", "pelapor_contact_id", "crm_contacts.csv"}, {"support_tickets.csv", "bug_id", "bugs.csv"}, {"support_tickets.csv", "versi_aplikasi", "releases.csv"}, {"bugs.csv", "versi_terdampak", "releases.csv"}, {"bugs.csv", "fitur_terkait", "features.csv"}}
	for _, j := range joins {
		for i, row := range s.rows[j.file] {
			v := row[j.col]
			if v != "" && !sets[j.target][v] {
				return fmt.Errorf("%s record %d: missing FK %s", j.file, i+1, j.col)
			}
		}
	}
	dateCols := map[string][]string{"crm_deals.csv": {"dibuat", "stage_sejak"}, "contact_employment_history.csv": {"mulai", "selesai"}, "interactions.jsonl": {"tanggal"}, "product_usage_daily.csv": {"tanggal"}, "decision_log.csv": {"tanggal"}, "support_tickets.csv": {"dibuat", "diselesaikan"}, "bugs.csv": {"dibuat", "selesai"}, "releases.csv": {"tanggal_rilis"}, "contracts_billing.csv": {"mulai", "tanggal_renewal"}}
	for file, cols := range dateCols {
		for i, row := range s.rows[file] {
			for _, col := range cols {
				if row[col] != "" {
					if _, err := models.ParseDate(row[col]); err != nil {
						return fmt.Errorf("%s record %d: invalid date %s", file, i+1, col)
					}
				}
			}
		}
	}
	nums := map[string][]string{"crm_deals.csv": {"outlet", "nilai_tahunan"}, "crm_accounts.csv": {"jumlah_outlet"}, "product_usage_daily.csv": {"jumlah_transaksi", "transaksi_offline_tersinkron"}, "feature_usage_monthly.csv": {"pengguna_aktif"}, "contracts_billing.csv": {"outlet_kontrak", "harga_per_outlet_bulan", "diskon_pct", "nilai_tahunan", "keterlambatan_bayar_12bln"}}
	for file, cols := range nums {
		for i, row := range s.rows[file] {
			for _, col := range cols {
				if row[col] != "" {
					n, err := strconv.ParseInt(row[col], 10, 64)
					if err != nil || n < 0 {
						return fmt.Errorf("%s record %d: invalid integer %s", file, i+1, col)
					}
				}
			}
		}
	}
	for _, row := range s.rows["contracts_billing.csv"] {
		if err := validatePackageLimit(row["batas_outlet_paket"]); err != nil {
			return err
		}
	}
	accounts := map[string]map[string]string{}
	outlets := map[string]string{}
	deals := map[string]string{}
	interactions := map[string]map[string]string{}
	decisions := map[string]map[string]string{}
	for _, x := range s.rows["crm_accounts.csv"] {
		accounts[x["account_id"]] = x
	}
	for _, x := range s.rows["outlets.csv"] {
		outlets[x["outlet_id"]] = x["account_id"]
	}
	for _, x := range s.rows["crm_deals.csv"] {
		deals[x["deal_id"]] = x["account_id"]
	}
	for _, x := range s.rows["interactions.jsonl"] {
		interactions[x["interaction_id"]] = x
	}
	for _, x := range s.rows["decision_log.csv"] {
		decisions[x["decision_id"]] = x
	}
	for _, x := range s.rows["interactions.jsonl"] {
		for _, id := range strings.Split(x["peserta"], ";") {
			if id != "" && !sets["crm_contacts.csv"][id] && !sets["employees.csv"][id] {
				return fmt.Errorf("unknown native participant")
			}
		}
		if p := x["membalas_id"]; p != "" {
			parent := interactions[p]
			if parent["tanggal"] > x["tanggal"] || parent["account_id"] != x["account_id"] {
				return fmt.Errorf("invalid reply scope/time")
			}
		}
	}
	for _, file := range []string{"product_usage_daily.csv", "support_tickets.csv"} {
		for _, x := range s.rows[file] {
			if x["outlet_id"] != "" && outlets[x["outlet_id"]] != x["account_id"] {
				return fmt.Errorf("outlet/account mismatch")
			}
			if file == "product_usage_daily.csv" && accounts[x["account_id"]]["tipe"] != "pelanggan" {
				return fmt.Errorf("usage attributed to prospect")
			}
		}
	}
	for _, x := range s.rows["decision_log.csv"] {
		if x["deal_id"] != "" && deals[x["deal_id"]] != x["account_id"] {
			return fmt.Errorf("decision/deal mismatch")
		}
		if i := x["bukti_interaction_id"]; i != "" {
			p := interactions[i]
			if p["account_id"] != x["account_id"] || p["tanggal"] > x["tanggal"] {
				return fmt.Errorf("decision/evidence mismatch")
			}
		}
	}
	for _, x := range s.rows["contracts_billing.csv"] {
		if id := x["decision_id"]; id != "" && decisions[id]["account_id"] != x["account_id"] {
			return fmt.Errorf("contract/decision mismatch")
		}
	}
	for _, x := range s.rows["feature_usage_monthly.csv"] {
		if _, err := time.Parse("2006-01", x["bulan"]); err != nil {
			return fmt.Errorf("invalid usage month")
		}
	}
	return nil
}

func validatePackageLimit(v string) error {
	if v == "tanpa batas" {
		return nil
	}
	n, err := strconv.Atoi(v)
	if err != nil || n < 1 {
		return fmt.Errorf("invalid package limit")
	}
	return nil
}
