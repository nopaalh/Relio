package main

import "testing"

func TestFullActionTaxonomyOnlyNativeDecisionValues(t *testing.T) {
	for _, c := range []struct{ typ, value, id string }{{"diskon", "15%", "action:discount:1500bps"}, {"pengecualian", "Paket Starter tanpa diskon, pilot 6 outlet", "action:starter_pilot"}, {"diskon", "unknown", ""}, {"janji_fitur", "custom promise", "action:feature_promise"}, {"buyer_request", "referensi", ""}} {
		id, _, _, bps := classifyDecisionAction(map[string]string{"tipe": c.typ, "nilai": c.value})
		if id != c.id {
			t.Fatalf("%s %s: %s", c.typ, c.value, id)
		}
		if c.id == "action:discount:1500bps" && (bps == nil || *bps != 1500) {
			t.Fatal("percent not converted to BPS")
		}
	}
}
