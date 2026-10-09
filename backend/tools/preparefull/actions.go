package main

import (
	"encoding/json"
	"fmt"
	"github.com/nopaalh/Relio/backend/models"
	"regexp"
	"sort"
	"strconv"
	"strings"
)

var discountValue = regexp.MustCompile(`^([0-9]+)%$`)

func classifyDecisionAction(row map[string]string) (id, typ, definition string, bps *int32) {
	switch row["tipe"] {
	case "diskon":
		m := discountValue.FindStringSubmatch(strings.TrimSpace(row["nilai"]))
		if m == nil {
			return
		}
		n, e := strconv.Atoi(m[1])
		if e != nil || n < 0 || n > 100 {
			return
		}
		v := int32(n * 100)
		bps = &v
		id = fmt.Sprintf("action:discount:%dbps", v)
		typ = "discount"
		definition = "Preseden keputusan diskon " + row["nilai"] + "; bukan izin permintaan baru."
	case "pengecualian":
		if strings.Contains(strings.ToLower(row["nilai"]), "pilot") && strings.Contains(strings.ToLower(row["nilai"]), "starter") {
			id = "action:starter_pilot"
			typ = "starter_pilot"
			definition = "Preseden pengecualian paket Starter untuk pilot, tidak mengasumsikan aplikasi/hasil."
		}
	case "janji_fitur":
		id = "action:feature_promise"
		typ = "feature_promise"
		definition = "Preseden keputusan janji fitur; tidak menyatakan fitur sudah tersedia."
	case "eskalasi":
		id = "action:escalation"
		typ = "escalation"
		definition = "Preseden keputusan eskalasi, tanpa mengasumsikan resolved."
	}
	return
}
func (b *builder) actions() {
	templates := map[string]models.ActionTemplate{}
	scopes := map[string]map[string]bool{}
	dates := map[string]models.Date{}
	for _, row := range b.src.rows["decision_log.csv"] {
		id, typ, definition, bps := classifyDecisionAction(row)
		if id == "" {
			continue
		}
		a, deal, native, at := row["account_id"], row["deal_id"], row["decision_id"], date(row["tanggal"])
		proof := func(field string) string { return "ev:decision_log.csv:" + native + ":" + field }
		status, ok := map[string]string{"Disetujui": "approved", "Ditolak": "rejected", "Menunggu": "requested"}[row["keputusan"]]
		if !ok {
			continue
		}
		actor := models.ParticipantRef{VerificationState: "unresolved", MatchMethod: "missing", CandidateNodeIDs: []string{}, EvidenceIDs: []string{}}
		if row["diminta_oleh"] != "" {
			actor = nativeParticipant(row["diminta_oleh"], proof("diminta_oleh"))
		}
		approver := models.ParticipantRef{VerificationState: "unresolved", MatchMethod: "missing", CandidateNodeIDs: []string{}, EvidenceIDs: []string{}}
		if row["diputuskan_oleh"] != "" {
			approver = nativeParticipant(row["diputuskan_oleh"], proof("diputuskan_oleh"))
		}
		p := models.PolicyFacts{DecisionID: ptr(native), RequestedDiscountBPS: unknown[int32]("no source-supported percent request"), ApprovedDiscountBPS: unknown[int32]("not approved or no percentage"), ApprovalStatus: known(status, "event", proof("keputusan")), Approver: approver, ConsentStatus: unknown[string]("no reference consent evidence"), ConsentScope: unknown[string]("no reference consent scope evidence")}
		if bps != nil {
			if row["diminta_oleh"] != "" {
				p.RequestedDiscountBPS = known(*bps, "event", proof("nilai"), proof("diminta_oleh"))
			}
			if status == "approved" {
				p.ApprovedDiscountBPS = known(*bps, "event", proof("nilai"), proof("keputusan"))
			}
		}
		occid := "occ:decision:" + native
		var ev models.Event
		_ = json.Unmarshal(b.records["events|event:decision:"+native].Payload, &ev)
		o := models.ActionOccurrence{OccurrenceID: occid, ActionID: id, EventID: ev.EventID, Actor: actor, Targets: []models.ParticipantRef{}, EventAt: at, Status: known(status, "event", proof("keputusan")), RawStatus: ptr(row["keputusan"]), OutcomeObserved: unknown[string]("no dated outcome evidence"), AccountIDs: []string{a}, DealIDs: []string{}, EvidenceIDs: []string{proof("decision_id"), proof("tanggal"), proof("tipe"), proof("nilai"), proof("keputusan")}, NodeIDs: append([]string{}, ev.NodeIDs...), EdgeIDs: append([]string{}, ev.EdgeIDs...), Policy: p}
		if actor.NodeID != nil {
			o.EvidenceIDs = append(o.EvidenceIDs, actor.EvidenceIDs...)
		}
		if approver.NodeID != nil {
			o.EvidenceIDs = append(o.EvidenceIDs, approver.EvidenceIDs...)
		}
		if deal != "" {
			o.DealIDs = []string{deal}
			o.EvidenceIDs = append(o.EvidenceIDs, proof("deal_id"))
		}
		sort.Strings(o.EvidenceIDs)
		b.put("occurrences", occid, a, deal, at, o)
		for _, eid := range o.EvidenceIDs {
			b.linkProof(eid, "", "", occid)
		}
		templates[id] = models.ActionTemplate{ActionID: id, ActionType: typ, Definition: definition, TaxonomyVersion: "relio-actions-v1"}
		if scopes[id] == nil {
			scopes[id] = map[string]bool{}
		}
		scopes[id][a] = true
		if dates[id] == "" || at < dates[id] {
			dates[id] = at
		}
	}
	for id, t := range templates {
		accounts := []string{}
		for a := range scopes[id] {
			accounts = append(accounts, a)
		}
		sort.Strings(accounts)
		b.put("templates", id, accounts[0], "", dates[id], t)
		r := b.records["templates|"+id]
		r.AccountIDs = accounts
		b.records["templates|"+id] = r
	}
}
