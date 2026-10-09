package main

import (
	"encoding/json"
	"fmt"
	"github.com/nopaalh/Relio/backend/models"
	"github.com/nopaalh/Relio/backend/repository"
	"sort"
	"strconv"
	"strings"
)

type builder struct {
	src         sources
	records     map[string]repository.FullRecord
	nodes       map[string]models.Node
	scope       map[string]map[string]bool
	from        map[string]models.Date
	sourceParts []repository.FullSourcePartition
}

func ptr[T any](v T) *T { return &v }
func known[T any](v T, basis string, eids ...string) models.Fact[T] {
	return models.Fact[T]{Value: ptr(v), State: "known", TemporalBasis: basis, EvidenceIDs: eids, Limitations: []string{}}
}
func unknown[T any](reason string) models.Fact[T] {
	return models.Fact[T]{State: "unknown", TemporalBasis: "undated", EvidenceIDs: []string{}, Limitations: []string{reason}}
}
func date(s string) models.Date { return models.Date(s) }
func (b *builder) put(kind, id, account, deal string, at models.Date, v any) {
	payload, err := json.Marshal(v)
	if err != nil {
		panic(err)
	}
	accounts, deals := []string{}, []string{}
	if account != "" {
		accounts = append(accounts, account)
	}
	if deal != "" {
		deals = append(deals, deal)
	}
	scope := account
	if scope == "" {
		scope = "company"
	}
	b.records[kind+"|"+id] = repository.FullRecord{Kind: kind, ID: id, PartitionID: scope + "|" + string(at)[:7] + "|" + kind, AccountIDs: accounts, DealIDs: deals, AvailableFrom: at, Payload: payload}
}
func (b *builder) evidence(file string, row map[string]string, key map[string]string, nativeID, field, account, deal string, available models.Date, basis string) *models.Evidence {
	locator := nativeID
	kind := "native"
	if locator == "" {
		locator = repository.FullDigestJSON(key)
		kind = "composite"
	}
	id := "ev:" + file + ":" + locator + ":" + field
	e := models.Evidence{EvidenceID: id, SourceFile: file, SourceRecordID: nativeID, RecordIDKind: kind, SourceKey: key, SourceChecksum: b.src.hashes[file], SourceField: field, SourceDate: nil, TemporalBasis: basis, ContentExcerpt: row[field], ScopeKind: "company", VerificationState: "verified", DealIDs: []string{}, AccountIDs: []string{}, EventIDs: []string{}, EdgeIDs: []string{}, OccurrenceIDs: []string{}}
	if account != "" {
		e.ScopeKind = "account"
		e.AccountIDs = []string{account}
	}
	if deal != "" {
		e.ScopeKind = "deal"
		e.DealIDs = []string{deal}
	}
	if basis == "event" || basis == "interval" {
		e.SourceDate = ptr(available)
	}
	b.put("evidence", id, account, deal, available, e)
	return &e
}
func (b *builder) fields(file string, row map[string]string, idcol, account, deal string, at models.Date, basis string) map[string]string {
	key := map[string]string{idcol: row[idcol]}
	out := map[string]string{}
	fields := []string{}
	for f := range row {
		fields = append(fields, f)
	}
	sort.Strings(fields)
	for _, f := range fields {
		e := b.evidence(file, row, key, row[idcol], f, account, deal, at, basis)
		out[f] = e.EvidenceID
	}
	return out
}
func (b *builder) node(id, typ, entity, account string, at models.Date, label models.Fact[string], eids []string) {
	n, ok := b.nodes[id]
	if !ok {
		n = models.Node{NodeID: id, NodeType: typ, EntityID: entity, Label: label, Validity: models.Validity{TemporalBasis: "event"}, VerificationState: "verified", EvidenceIDs: []string{}, EventIDs: []string{}}
	}
	n.EvidenceIDs = append(n.EvidenceIDs, eids...)
	b.nodes[id] = n
	if b.scope[id] == nil {
		b.scope[id] = map[string]bool{}
	}
	if account != "" {
		b.scope[id][account] = true
	} else {
		b.scope[id]["company"] = true
	}
	if b.from[id] == "" || at < b.from[id] {
		b.from[id] = at
	}
}
func (b *builder) edge(typ, src, target, account, deal string, at models.Date, evs, events []string, valid models.Validity) string {
	id := "edge:" + repository.FullDigestJSON([]any{typ, src, target, evs, events})
	e := models.Edge{EdgeID: id, EdgeType: typ, Source: src, Target: target, Validity: valid, VerificationState: "verified", MatchMethod: "native_fk", EvidenceIDs: evs, EventIDs: events}
	b.put("edges", id, account, deal, at, e)
	return id
}
func emptyParticipant() models.ParticipantRef {
	return models.ParticipantRef{VerificationState: "unresolved", MatchMethod: "no_source_identity", CandidateNodeIDs: []string{}, EvidenceIDs: []string{}}
}
func nativeParticipant(id, ev string) models.ParticipantRef {
	typ := "contact:"
	if strings.HasPrefix(id, "E") {
		typ = "employee:"
	}
	return models.ParticipantRef{NodeID: ptr(typ + id), RawRef: ptr(id), VerificationState: "verified", MatchMethod: "native_id", CandidateNodeIDs: []string{}, EvidenceIDs: []string{ev}}
}
func (b *builder) emailParticipant(raw, account, ev string, at models.Date) models.ParticipantRef {
	p := emptyParticipant()
	if raw != "" {
		p.RawRef = ptr(raw)
		p.EvidenceIDs = []string{ev}
		p.MatchMethod = "raw_email_no_historical_alias"
	}
	for _, file := range []string{"crm_contacts.csv", "employees.csv"} {
		for _, row := range b.src.rows[file] {
			if row["email"] == raw && raw != "" {
				col, typ := "contact_id", "contact:"
				if file == "employees.csv" {
					col = "employee_id"
					typ = "employee:"
				}
				id := typ + row[col]
				p.CandidateNodeIDs = append(p.CandidateNodeIDs, id)
				p.VerificationState = "ambiguous"
				b.node(id, "person", row[col], account, at, known(row[col], "event", ev), []string{ev})
			}
		}
	}
	return p
}
func (b *builder) buildCore() {
	for _, row := range b.src.rows["crm_accounts.csv"] {
		a := row["account_id"]
		ev := b.fields("crm_accounts.csv", row, "account_id", a, "", repository.MaxAsOf, "snapshot")
		b.node("account:"+a, "account", a, a, repository.MaxAsOf, known(row["nama"], "snapshot", ev["nama"]), []string{ev["account_id"], ev["nama"]})
	}
	for _, file := range []string{"crm_contacts.csv", "employees.csv"} {
		for _, row := range b.src.rows[file] {
			col, typ, account := "contact_id", "contact:", row["account_id_saat_ini"]
			if file == "employees.csv" {
				col = "employee_id"
				typ = "employee:"
			}
			ev := b.fields(file, row, col, account, "", repository.MaxAsOf, "snapshot")
			b.node(typ+row[col], "person", row[col], account, repository.MaxAsOf, known(row["nama"], "snapshot", ev["nama"]), []string{ev[col], ev["nama"]})
		}
	}
	for _, row := range b.src.rows["crm_deals.csv"] {
		id, a, at := row["deal_id"], row["account_id"], date(row["dibuat"])
		ev := b.fields("crm_deals.csv", row, "deal_id", a, id, repository.MaxAsOf, "snapshot")
		for _, f := range []string{"deal_id", "account_id", "dibuat"} {
			ev[f] = b.evidence("crm_deals.csv", row, map[string]string{"deal_id": id}, id, f, a, id, at, "event").EvidenceID
		}
		outlets, _ := strconv.Atoi(row["outlet"])
		acv, _ := strconv.ParseInt(row["nilai_tahunan"], 10, 64)
		d := models.DealFacts{DealID: id, AccountID: a, DealType: known(row["tipe"], "snapshot", ev["tipe"]), OwnerID: known(row["owner_id"], "snapshot", ev["owner_id"]), Stage: known(row["stage"], "snapshot", ev["stage"]), StageSince: known(date(row["stage_sejak"]), "snapshot", ev["stage_sejak"]), CreatedAt: known(at, "event", ev["dibuat"]), PlannedOutlets: known(outlets, "snapshot", ev["outlet"]), PotentialACVIDR: known(acv, "snapshot", ev["nilai_tahunan"]), Status: known(row["status"], "snapshot", ev["status"]), RecordEvidenceIDs: []string{}}
		for _, v := range ev {
			d.RecordEvidenceIDs = append(d.RecordEvidenceIDs, v)
		}
		sort.Strings(d.RecordEvidenceIDs)
		b.put("deals", id, a, id, at, d)
		b.node("deal:"+id, "deal", id, a, at, known(id, "event", ev["deal_id"]), []string{ev["deal_id"]})
		b.node("account:"+a, "account", a, a, at, known(a, "event", ev["account_id"]), []string{ev["account_id"]})
		b.edge("DEAL_OF_ACCOUNT", "deal:"+id, "account:"+a, a, id, at, []string{ev["account_id"]}, []string{}, models.Validity{TemporalBasis: "event"})
		b.edge("OWNED_BY", "deal:"+id, "employee:"+row["owner_id"], a, id, repository.MaxAsOf, []string{ev["owner_id"]}, []string{}, models.Validity{TemporalBasis: "snapshot"})
		b.node("employee:"+row["owner_id"], "person", row["owner_id"], a, repository.MaxAsOf, known(row["owner_id"], "snapshot", ev["owner_id"]), []string{ev["owner_id"]})
	}
	for _, row := range b.src.rows["outlets.csv"] {
		a, id := row["account_id"], row["outlet_id"]
		ev := b.fields("outlets.csv", row, "outlet_id", a, "", repository.MaxAsOf, "snapshot")
		b.node("outlet:"+id, "outlet", id, a, repository.MaxAsOf, known(id, "snapshot", ev["outlet_id"]), []string{ev["outlet_id"]})
		b.edge("HAS_OUTLET", "account:"+a, "outlet:"+id, a, "", repository.MaxAsOf, []string{ev["account_id"]}, []string{}, models.Validity{TemporalBasis: "snapshot"})
	}
	for _, row := range b.src.rows["contact_employment_history.csv"] {
		a, at := row["account_id"], date(row["mulai"])
		key := map[string]string{}
		for k, v := range row {
			key[k] = v
		}
		eid := "employment:" + repository.FullDigestJSON(key)
		org := "account:" + a
		if a == "" {
			org = "organization:" + repository.FullDigestJSON(row["organisasi"])
		}
		evs := []string{}
		role := ""
		for _, f := range []string{"contact_id", "account_id", "organisasi", "jabatan", "mulai", "selesai"} {
			available := at
			if f == "selesai" && row[f] != "" {
				available = date(row[f])
			}
			e := b.evidence("contact_employment_history.csv", row, key, "", f, a, "", available, "interval")
			evs = append(evs, e.EvidenceID)
			if f == "jabatan" {
				role = e.EvidenceID
			}
		}
		b.node("contact:"+row["contact_id"], "person", row["contact_id"], a, at, known(row["contact_id"], "interval", evs[0]), []string{evs[0]})
		b.node(org, "organization", row["organisasi"], a, at, known(row["organisasi"], "interval", evs[2]), []string{evs[2]})
		valid := models.Validity{ValidFrom: ptr(at), TemporalBasis: "interval"}
		if row["selesai"] != "" {
			end, _ := date(row["selesai"]).NextDay()
			valid.ValidTo = &end
		} else {
			valid.OpenEndReason = ptr("source_end_missing")
		}
		b.edge("WORKED_AT", "contact:"+row["contact_id"], org, a, "", at, evs, []string{eid}, valid)
		e := models.Event{EventID: eid, VerificationState: "verified", EventAt: at, TimePrecision: "date", EventType: "CONTACT_TRANSITION", Status: unknown[string]("employment interval, not employment approval"), Summary: known(row["jabatan"], "interval", role), Actors: []models.ParticipantRef{}, Targets: []models.ParticipantRef{}, Participants: []models.ParticipantRef{nativeParticipant(row["contact_id"], evs[0])}, ScopeKind: "account", AccountIDs: []string{a}, DealIDs: []string{}, NodeIDs: []string{"contact:" + row["contact_id"], org}, EdgeIDs: []string{}, EvidenceIDs: evs}
		if a == "" {
			e.ScopeKind = "company"
			e.AccountIDs = []string{}
		}
		b.put("events", eid, a, "", at, e)
	}
}
func (b *builder) interactions() {
	for _, row := range b.src.rows["interactions.jsonl"] {
		a, id, at := row["account_id"], row["interaction_id"], date(row["tanggal"])
		ev := b.fields("interactions.jsonl", row, "interaction_id", a, "", at, "event")
		eid := "event:interaction:" + id
		e := models.Event{EventID: eid, VerificationState: "verified", EventAt: at, TimePrecision: "date", EventType: row["tipe"], Status: unknown[string]("no native interaction status"), Summary: known(row["subjek"], "event", ev["subjek"]), Actors: []models.ParticipantRef{b.emailParticipant(row["dari"], a, ev["dari"], at)}, Targets: []models.ParticipantRef{}, Participants: []models.ParticipantRef{}, ScopeKind: "account", AccountIDs: []string{a}, DealIDs: []string{}, NodeIDs: []string{eid}, EdgeIDs: []string{}, EvidenceIDs: []string{}}
		if a == "" {
			e.ScopeKind = "company"
			e.AccountIDs = []string{}
		} else {
			b.node("account:"+a, "account", a, a, at, known(a, "event", ev["account_id"]), []string{ev["account_id"]})
			e.NodeIDs = append(e.NodeIDs, "account:"+a)
			e.EdgeIDs = append(e.EdgeIDs, b.edge("CONCERNS_ACCOUNT", eid, "account:"+a, a, "", at, []string{ev["account_id"]}, []string{eid}, models.Validity{TemporalBasis: "event"}))
		}
		for _, email := range strings.Split(row["ke"], ";") {
			if email != "" {
				e.Targets = append(e.Targets, b.emailParticipant(email, a, ev["ke"], at))
			}
		}
		for _, pid := range strings.Split(row["peserta"], ";") {
			if pid != "" {
				p := nativeParticipant(pid, ev["peserta"])
				e.Participants = append(e.Participants, p)
				e.NodeIDs = append(e.NodeIDs, *p.NodeID)
				b.node(*p.NodeID, "person", pid, a, at, known(pid, "event", ev["peserta"]), []string{ev["peserta"]})
				e.EdgeIDs = append(e.EdgeIDs, b.edge("PARTICIPATED_IN", *p.NodeID, eid, a, "", at, []string{ev["peserta"]}, []string{eid}, models.Validity{TemporalBasis: "event"}))
			}
		}
		for _, v := range ev {
			e.EvidenceIDs = append(e.EvidenceIDs, v)
		}
		sort.Strings(e.EvidenceIDs)
		if row["membalas_id"] != "" {
			e.EdgeIDs = append(e.EdgeIDs, b.edge("REPLIES_TO", eid, "event:interaction:"+row["membalas_id"], a, "", at, []string{ev["membalas_id"]}, []string{eid}, models.Validity{TemporalBasis: "event"}))
		}
		b.node(eid, "event", id, a, at, e.Summary, e.EvidenceIDs)
		n := b.nodes[eid]
		n.EventIDs = []string{eid}
		b.nodes[eid] = n
		b.put("events", eid, a, "", at, e)
	}
}
func (b *builder) finish() ([]repository.FullRecord, error) {
	for _, r := range b.records {
		if r.Kind == "events" {
			var e models.Event
			_ = json.Unmarshal(r.Payload, &e)
			for _, id := range e.NodeIDs {
				n := b.nodes[id]
				n.EventIDs = append(n.EventIDs, e.EventID)
				b.nodes[id] = n
			}
			for _, id := range e.EvidenceIDs {
				b.linkProof(id, e.EventID, "", "")
			}
		}
		if r.Kind == "edges" {
			var e models.Edge
			_ = json.Unmarshal(r.Payload, &e)
			for _, id := range e.EvidenceIDs {
				b.linkProof(id, "", e.EdgeID, "")
			}
		}
	}
	for id, n := range b.nodes {
		sort.Strings(n.EvidenceIDs)
		n.EvidenceIDs = dedup(n.EvidenceIDs)
		sort.Strings(n.EventIDs)
		n.EventIDs = dedup(n.EventIDs)
		accounts := []string{}
		for a := range b.scope[id] {
			if a != "company" {
				accounts = append(accounts, a)
			}
		}
		sort.Strings(accounts)
		at := b.from[id]
		account := ""
		if len(accounts) > 0 {
			account = accounts[0]
		}
		b.put("nodes", id, account, "", at, n)
		r := b.records["nodes|"+id]
		r.AccountIDs = accounts
		b.records["nodes|"+id] = r
	}
	records := []repository.FullRecord{}
	for _, r := range b.records {
		records = append(records, r)
	}
	return records, nil
}

func (b *builder) linkProof(id, event, edge, occurrence string) {
	key := "evidence|" + id
	r, ok := b.records[key]
	if !ok {
		return
	}
	var e models.Evidence
	_ = json.Unmarshal(r.Payload, &e)
	if event != "" {
		e.EventIDs = append(e.EventIDs, event)
	}
	if edge != "" {
		e.EdgeIDs = append(e.EdgeIDs, edge)
	}
	if occurrence != "" {
		e.OccurrenceIDs = append(e.OccurrenceIDs, occurrence)
	}
	sort.Strings(e.EventIDs)
	e.EventIDs = dedup(e.EventIDs)
	sort.Strings(e.EdgeIDs)
	e.EdgeIDs = dedup(e.EdgeIDs)
	sort.Strings(e.OccurrenceIDs)
	e.OccurrenceIDs = dedup(e.OccurrenceIDs)
	r.Payload, _ = json.Marshal(e)
	b.records[key] = r
}
func dedup(xs []string) []string {
	ys := []string{}
	for _, x := range xs {
		if len(ys) == 0 || ys[len(ys)-1] != x {
			ys = append(ys, x)
		}
	}
	return ys
}
func build(s sources) (repository.FullArtifact, error) {
	b := &builder{src: s, records: map[string]repository.FullRecord{}, nodes: map[string]models.Node{}, scope: map[string]map[string]bool{}, from: map[string]models.Date{}, sourceParts: []repository.FullSourcePartition{}}
	b.buildCore()
	b.interactions()
	b.business()
	b.usage()
	records, err := b.finish()
	if err != nil {
		return repository.FullArtifact{}, err
	}
	version := "full:" + repository.FullDigestJSON([]any{s.hashes, repository.FullSchemaVersion, "mapping-v1", "relio-actions-v1", "relio-relevance-v1"})
	a := repository.FullArtifact{Manifest: repository.FullManifest{SchemaVersion: repository.FullSchemaVersion, DatasetVersion: version, Sources: s.hashes, SourceCounts: s.counts, DealIDs: []string{}, AccountIDs: []string{}}, Records: records, SourcePartitions: b.sourceParts}
	for _, x := range s.rows["crm_deals.csv"] {
		a.Manifest.DealIDs = append(a.Manifest.DealIDs, x["deal_id"])
	}
	for _, x := range s.rows["crm_accounts.csv"] {
		a.Manifest.AccountIDs = append(a.Manifest.AccountIDs, x["account_id"])
	}
	if err := repository.SealFullArtifact(&a); err != nil {
		return a, fmt.Errorf("seal: %w", err)
	}
	return a, validateGraph(a)
}
