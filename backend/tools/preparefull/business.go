package main

import (
	"github.com/nopaalh/Relio/backend/models"
	"github.com/nopaalh/Relio/backend/repository"
	"sort"
	"strings"
)

func (b *builder) business() {
	for _, row := range b.src.rows["decision_log.csv"] {
		a, id, deal, at := row["account_id"], row["decision_id"], row["deal_id"], date(row["tanggal"])
		ev := b.fields("decision_log.csv", row, "decision_id", a, deal, at, "event")
		for _, field := range []string{"alasan", "status_janji"} {
			ev[field] = b.evidence("decision_log.csv", row, map[string]string{"decision_id": id}, id, field, a, deal, repository.MaxAsOf, "snapshot").EvidenceID
		}
		eid := "event:decision:" + id
		scope := "account"
		if deal != "" {
			scope = "deal"
		}
		e := models.Event{EventID: eid, VerificationState: "verified", EventAt: at, TimePrecision: "date", EventType: "COMMERCIAL_DECISION", Status: known(row["keputusan"], "event", ev["keputusan"]), RawStatus: ptr(row["keputusan"]), Summary: known(row["tipe"]+": "+row["nilai"], "event", ev["tipe"], ev["nilai"]), Actors: []models.ParticipantRef{}, Targets: []models.ParticipantRef{}, Participants: []models.ParticipantRef{}, ScopeKind: scope, AccountIDs: []string{a}, DealIDs: []string{}, NodeIDs: []string{eid, "decision:" + id, "account:" + a}, EdgeIDs: []string{}, EvidenceIDs: []string{}}
		if deal != "" {
			e.DealIDs = []string{deal}
			e.NodeIDs = append(e.NodeIDs, "deal:"+deal)
		}
		for _, field := range []string{"diminta_oleh", "diputuskan_oleh"} {
			if row[field] != "" {
				p := nativeParticipant(row[field], ev[field])
				if field == "diputuskan_oleh" {
					e.Actors = append(e.Actors, p)
				} else {
					e.Participants = append(e.Participants, p)
				}
				b.node(*p.NodeID, "person", row[field], a, at, known(row[field], "event", ev[field]), []string{ev[field]})
				e.NodeIDs = append(e.NodeIDs, *p.NodeID)
				edgeType := "REQUESTED_BY"
				if field == "diputuskan_oleh" {
					edgeType = "DECIDED_BY"
				}
				e.EdgeIDs = append(e.EdgeIDs, b.edge(edgeType, "decision:"+id, *p.NodeID, a, deal, at, []string{ev[field]}, []string{eid}, models.Validity{TemporalBasis: "event"}))
			}
		}
		for _, v := range ev {
			e.EvidenceIDs = append(e.EvidenceIDs, v)
		}
		sort.Strings(e.EvidenceIDs)
		b.node("account:"+a, "account", a, a, at, known(a, "event", ev["account_id"]), []string{ev["account_id"]})
		b.node("decision:"+id, "decision", id, a, at, e.Summary, []string{ev["decision_id"], ev["tipe"], ev["nilai"]})
		b.node(eid, "event", id, a, at, e.Summary, e.EvidenceIDs)
		e.EdgeIDs = append(e.EdgeIDs, b.edge("CONCERNS_ACCOUNT", eid, "account:"+a, a, deal, at, []string{ev["account_id"]}, []string{eid}, models.Validity{TemporalBasis: "event"}))
		e.EdgeIDs = append(e.EdgeIDs, b.edge("HAS_DECISION", eid, "decision:"+id, a, deal, at, []string{ev["decision_id"]}, []string{eid}, models.Validity{TemporalBasis: "event"}))
		if deal != "" {
			e.EdgeIDs = append(e.EdgeIDs, b.edge("CONCERNS_DEAL", eid, "deal:"+deal, a, deal, at, []string{ev["deal_id"]}, []string{eid}, models.Validity{TemporalBasis: "event"}))
		}
		if row["bukti_interaction_id"] != "" {
			e.EdgeIDs = append(e.EdgeIDs, b.edge("SUPPORTED_BY", "decision:"+id, "event:interaction:"+row["bukti_interaction_id"], a, deal, at, []string{ev["bukti_interaction_id"]}, []string{eid}, models.Validity{TemporalBasis: "event"}))
		}
		if row["fitur_dijanjikan"] != "" {
			e.EdgeIDs = append(e.EdgeIDs, b.edge("PRODUCT_REQUIREMENT", "decision:"+id, "feature:"+row["fitur_dijanjikan"], a, deal, at, []string{ev["fitur_dijanjikan"]}, []string{eid}, models.Validity{TemporalBasis: "event"}))
		}
		b.put("events", eid, a, deal, at, e)
	}
	for _, row := range b.src.rows["contracts_billing.csv"] {
		a, id, at := row["account_id"], row["contract_id"], date(row["mulai"])
		ev := b.fields("contracts_billing.csv", row, "contract_id", a, "", repository.MaxAsOf, "snapshot")
		for _, f := range []string{"contract_id", "account_id", "mulai"} {
			ev[f] = b.evidence("contracts_billing.csv", row, map[string]string{"contract_id": id}, id, f, a, "", at, "event").EvidenceID
		}
		eid := "event:contract:" + id
		b.node("account:"+a, "account", a, a, at, known(a, "event", ev["account_id"]), []string{ev["account_id"]})
		b.node("contract:"+id, "contract", id, a, at, known(id, "event", ev["contract_id"]), []string{ev["contract_id"]})
		b.node(eid, "event", id, a, at, known(id, "event", ev["contract_id"]), []string{ev["contract_id"]})
		edge := b.edge("HAS_CONTRACT", "account:"+a, "contract:"+id, a, "", at, []string{ev["account_id"], ev["mulai"]}, []string{eid}, models.Validity{TemporalBasis: "event"})
		evs := []string{}
		for _, v := range ev {
			evs = append(evs, v)
		}
		sort.Strings(evs)
		e := models.Event{EventID: eid, VerificationState: "verified", EventAt: at, TimePrecision: "date", EventType: "CONTRACT_TERM", Status: unknown[string]("contract terms are snapshot-only"), Summary: known(id, "event", ev["contract_id"]), Actors: []models.ParticipantRef{}, Targets: []models.ParticipantRef{}, Participants: []models.ParticipantRef{}, ScopeKind: "account", DealIDs: []string{}, AccountIDs: []string{a}, NodeIDs: []string{eid, "contract:" + id, "account:" + a}, EdgeIDs: []string{edge}, EvidenceIDs: evs}
		if row["decision_id"] != "" {
			e.EdgeIDs = append(e.EdgeIDs, b.edge("APPLIED_DECISION", "contract:"+id, "decision:"+row["decision_id"], a, "", repository.MaxAsOf, []string{ev["decision_id"], ev["diskon_pct"]}, []string{eid}, models.Validity{TemporalBasis: "snapshot"}))
		}
		b.put("events", eid, a, "", at, e)
	}
	for _, file := range []string{"features.csv", "bugs.csv", "releases.csv", "support_tickets.csv"} {
		for _, row := range b.src.rows[file] {
			col, typ, label, a, at := "feature_id", "feature", row["nama"], "", repository.MaxAsOf
			switch file {
			case "bugs.csv":
				col = "bug_id"
				typ = "bug"
				label = row["judul"]
				at = date(row["dibuat"])
			case "releases.csv":
				col = "versi"
				typ = "release"
				label = row["versi"]
				at = date(row["tanggal_rilis"])
			case "support_tickets.csv":
				col = "ticket_id"
				typ = "ticket"
				label = row["judul"]
				a = row["account_id"]
				at = date(row["dibuat"])
			}
			id := row[col]
			ev := b.fields(file, row, col, a, "", repository.MaxAsOf, "snapshot")
			if file == "releases.csv" {
				ev = b.fields(file, row, col, a, "", at, "event")
			}
			anchor := col
			ev[anchor] = b.evidence(file, row, map[string]string{col: id}, id, anchor, a, "", at, "event").EvidenceID
			labelField := anchor
			labelBasis := "snapshot"
			if typ == "ticket" || typ == "bug" {
				labelField = "judul"
			}
			if typ == "feature" {
				labelField = "nama"
			}
			if typ == "release" {
				labelBasis = "event"
			}
			b.node(typ+":"+id, typ, id, a, at, known(label, labelBasis, ev[labelField]), []string{ev[anchor], ev[labelField]})
			if file == "features.csv" {
				continue
			}
			eid := "event:" + typ + ":" + id
			evs := []string{}
			for _, v := range ev {
				evs = append(evs, v)
			}
			sort.Strings(evs)
			summary := known(id, "event", ev[anchor])
			if file == "releases.csv" {
				summary = known(label, "event", ev[anchor])
			}
			b.node(eid, "event", id, a, at, summary, []string{ev[anchor]})
			e := models.Event{EventID: eid, VerificationState: "verified", EventAt: at, TimePrecision: "date", EventType: strings.ToUpper(typ), Status: unknown[string]("current status has no history"), Summary: summary, Actors: []models.ParticipantRef{}, Targets: []models.ParticipantRef{}, Participants: []models.ParticipantRef{}, ScopeKind: "company", AccountIDs: []string{}, DealIDs: []string{}, NodeIDs: []string{eid, typ + ":" + id}, EdgeIDs: []string{}, EvidenceIDs: evs}
			if a != "" {
				e.ScopeKind = "account"
				e.AccountIDs = []string{a}
				b.node("account:"+a, "account", a, a, at, known(a, "event", ev[anchor]), []string{ev[anchor]})
				e.NodeIDs = append(e.NodeIDs, "account:"+a)
				e.EdgeIDs = append(e.EdgeIDs, b.edge("CONCERNS_ACCOUNT", eid, "account:"+a, a, "", at, []string{ev[anchor]}, []string{eid}, models.Validity{TemporalBasis: "event"}))
			}
			e.EdgeIDs = append(e.EdgeIDs, b.edge("DESCRIBES", eid, typ+":"+id, a, "", at, []string{ev[anchor]}, []string{eid}, models.Validity{TemporalBasis: "event"}))
			for _, link := range []struct{ field, kind string }{{"bug_id", "bug"}, {"feature_id", "feature"}, {"fitur_terkait", "feature"}, {"versi_terdampak", "release"}, {"versi_aplikasi", "release"}, {"outlet_id", "outlet"}, {"pelapor_contact_id", "contact"}} {
				if row[link.field] != "" && link.field != col {
					e.EdgeIDs = append(e.EdgeIDs, b.edge("SOURCE_REFERENCE", typ+":"+id, link.kind+":"+row[link.field], a, "", repository.MaxAsOf, []string{ev[link.field]}, []string{eid}, models.Validity{TemporalBasis: "snapshot"}))
				}
			}
			b.put("events", eid, a, "", at, e)
		}
	}
}
