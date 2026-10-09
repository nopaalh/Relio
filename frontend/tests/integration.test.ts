import test, { type TestContext } from "node:test";
import assert from "node:assert/strict";
import type { Assessment, CanonicalEvent, CanonicalNode, DealFacts, Fact, FactState, GraphResult, SnapshotMeta, EvidenceResult } from "../src/lib/contracts.ts";
import { applyAssessment, assertMeta, consumePages, dealEvidenceIds, knownValue, toDeal, toEvent, toEvidence, toGraph } from "../src/lib/adapters.ts";
import { api, ApiError, optionalRead, request } from "../src/lib/client.ts";
import { eventNode, linkedEvent, nodeEventIds, solarSystemLayout } from "../src/lib/universe.ts";

// DATA UJI only: offline fixtures, never an application dataset or provider result.
const date = "2026-10-01", id = "DATA-UJI-DL004", account = "DATA-UJI-P04";
const meta: SnapshotMeta = {
  as_of: date, max_as_of: date, calendar_zone: "Asia/Jakarta", context_id: "DATA-UJI-context",
  dataset_version: "DATA-UJI-dataset", contract_version: "DATA-UJI-contract", data_state: "partial",
  limitations: ["DATA UJI: partial account scope"], unknowns: [{ path: "participants", reason: "DATA UJI: unresolved identity", reference_ids: ["DATA-UJI:person"] }],
};
const complete = { truncated: false, truncation_reasons: [], next_cursor: null };
const known = <T extends string | number>(value: T): Fact<T> => ({ value, state: "known", temporal_basis: "snapshot", evidence_ids: ["DATA-UJI:claim"], limitations: [] });
const absent = <T extends string | number>(state: FactState = "unknown"): Fact<T> => ({ value: null, state, temporal_basis: "snapshot", evidence_ids: [], limitations: ["DATA UJI: value withheld"] });
const facts: DealFacts = {
  deal_id: id, account_id: account, deal_type: known("DATA-UJI-prospect"), owner_id: absent("ambiguous"),
  stage: absent("snapshot_only"), stage_since: absent("snapshot_only"), created_at: known("2026-01-01"),
  potential_acv_idr: known(0), planned_outlets: known(0), status: absent(), record_evidence_ids: ["DATA-UJI:record"],
};
const event = (eventId = "DATA-UJI:ev1", eventDate = "2026-09-01"): CanonicalEvent => ({
  event_id: eventId, event_at: eventDate, time_precision: "date", source_timestamp: null, event_type: "DATA-UJI-request",
  verification_state: "inferred", status: absent("ambiguous"), raw_status: "DATA-UJI raw request", summary: known("DATA UJI source summary"),
  actors: [{ node_id: null, raw_ref: "DATA-UJI raw participant", verification_state: "ambiguous", match_method: "DATA-UJI-unresolved",
    candidate_node_ids: ["DATA-UJI:person-a", "DATA-UJI:person-b"], evidence_ids: ["DATA-UJI:actor-claim"] }],
  targets: [], participants: [], scope_kind: "account", deal_ids: [], account_ids: [account],
  node_ids: ["Event:" + eventId], edge_ids: ["DATA-UJI:edge"], evidence_ids: ["DATA-UJI:event-claim"],
});
const node = (nodeId: string, type: string, entity: string, refs: string[] = []): CanonicalNode => ({
  node_id: nodeId, node_type: type, entity_id: entity, label: known(entity), person_kind: null, person: null,
  validity: { valid_from: null, valid_to: null, open_end_reason: "DATA-UJI unknown interval", temporal_basis: "undated" },
  verification_state: "inferred", evidence_ids: ["DATA-UJI:claim"], event_ids: refs,
});
const graph: GraphResult = {
  meta, bounds: { truncated: true, truncation_reasons: ["DATA-UJI node budget"], next_cursor: null },
  nodes: [node("Deal:" + id, "Deal", id), node("Account:" + account, "Account", account),
    node("Event:DATA-UJI:ev1", "Event", "DATA-UJI:ev1", ["DATA-UJI:ev1"]),
    node("Event:DATA-UJI:ev2", "Event", "DATA-UJI:ev2", ["DATA-UJI:ev2"]),
    node("Person:DATA-UJI:locator", "Person", "DATA-UJI:locator", ["DATA-UJI:ev1", "DATA-UJI:ev2"])],
  edges: [{ edge_id: "DATA-UJI:edge", edge_type: "HAS_EVENT", source: "Account:" + account, target: "Event:DATA-UJI:ev1",
    validity: { valid_from: null, valid_to: null, open_end_reason: "DATA-UJI unknown", temporal_basis: "event" },
    source_timestamp: null, observed_at: null, recorded_at: null, verification_state: "inferred", match_method: "DATA-UJI-source",
    evidence_ids: ["DATA-UJI:event-claim"], event_ids: ["DATA-UJI:ev1"] }],
};
const evidence: EvidenceResult = {
  meta, evidence: {
    evidence_id: "DATA-UJI:event-claim", source_file: "DATA-UJI.jsonl", source_record_id: "DATA-UJI:locator", record_id_kind: "derived",
    source_key: { line: "DATA-UJI-line" }, source_checksum: "DATA-UJI-source-checksum", source_field: "DATA-UJI-field",
    span: { start: 0, end: 12, field_checksum: "DATA-UJI-field-checksum" }, source_date: "2026-09-01", source_timestamp: null,
    temporal_basis: "event", content_excerpt: "DATA UJI exact authorized excerpt", scope_kind: "account", verification_state: "ambiguous",
    deal_ids: [], account_ids: [account], event_ids: ["DATA-UJI:ev1"], edge_ids: [], occurrence_ids: [],
  },
};
const assessment: Assessment = {
  meta, status: "DATA-UJI-assessed", rubric_version: "DATA-UJI-rubric", explanation: "DATA UJI supplied readiness only", unknowns: [],
  readiness_100: 0, readiness_label: "DATA-UJI level zero", jev_raw_score: 0, jev_confidence: 0, evidence_ids: ["DATA-UJI:claim"], provider_model: "DATA-UJI-model",
};
function mockFetch(t: TestContext, handler: (url: URL, init?: RequestInit) => Response | Promise<Response>) {
  t.mock.method(globalThis, "fetch", (input: RequestInfo | URL, init?: RequestInit) => handler(new URL(String(input), "http://data-uji.invalid"), init));
}
function coreReply(url: URL): Response | undefined {
  if (url.pathname === "/api/deals/" + id) return Response.json({ meta, deal: facts });
  if (url.pathname.endsWith("/timeline")) return Response.json({ meta, bounds: complete, events: [event()] });
  if (url.pathname.endsWith("/graph")) return Response.json(graph);
}
const unavailable = () => Response.json({ code: "DATA-UJI-unavailable", message: "DATA UJI provider unavailable" }, { status: 503 });

test("DATA UJI: known zero, unresolved facts and original provenance survive mapping", () => {
  assert.equal(knownValue(known(0)), 0);
  for (const state of ["unknown", "ambiguous", "snapshot_only"] as const) assert.equal(knownValue(absent(state)), null);
  const view = toDeal(facts, meta);
  assert.equal(view.acv, 0); assert.equal(view.outlets, 0); assert.equal(view.stage, null); assert.equal(view.owner_id, null);
  assert.equal(view.name, account); assert.equal(view.city, null); assert.equal(view.industry, null);
  for (const field of ["attractiveness", "coverage", "urgency", "readiness", "action_count"] as const) assert.equal(view[field], null);
  assert.strictEqual(view.facts, facts); assert.strictEqual(view.meta, meta);
  assert.deepEqual(view.record_evidence_ids, facts.record_evidence_ids);
  assert.ok(view.unknowns?.some(u => u.path === "coverage" && u.reason.includes("not a coverage score")));
  assert.ok(view.unknowns?.some(u => u.path === "participants" && u.reference_ids[0] === "DATA-UJI:person"));
  assert.deepEqual(dealEvidenceIds(view), ["DATA-UJI:record", "DATA-UJI:claim"]);
  assert.ok(dealEvidenceIds(view).every(ref => !ref.startsWith("EVC-")));
});

test("DATA UJI: contradictory facts cannot silently become known values", () => {
  assert.throws(() => knownValue({ ...absent<number>(), value: 0 }), /Unresolved/);
  assert.throws(() => knownValue({ ...known(0), evidence_ids: [] }), /provenance/);
  assert.throws(() => knownValue(known(Number.MAX_SAFE_INTEGER + 1)), /supported value/);
});

test("DATA UJI: historical snapshot withholding and structured limitations remain intact", () => {
  const historical = { ...meta, as_of: "2026-09-01", context_id: "DATA-UJI-historical" };
  const source = { ...facts, potential_acv_idr: absent<number>("snapshot_only"), planned_outlets: absent<number>("snapshot_only") };
  const view = toDeal(source, historical);
  assert.equal(view.acv, null); assert.equal(view.outlets, null);
  assert.equal(view.facts?.potential_acv_idr.state, "snapshot_only");
  assert.ok(view.unknowns?.some(u => u.path === "potential_acv_idr" && u.reason.includes("withheld")));
});

test("DATA UJI: account events keep ambiguous participants and do not acquire deal/interaction IDs", () => {
  const source = event(), view = toEvent(source, date);
  assert.strictEqual(view.source, source); assert.deepEqual(view.source?.deal_ids, []);
  assert.equal(view.source?.scope_kind, "account"); assert.deepEqual(view.source?.account_ids, [account]);
  assert.equal(view.status, null); assert.equal(view.interaction_id, null); assert.equal(view.actor_email, null);
  assert.match(view.actor, /ambiguous/); assert.equal(view.source?.actors[0].node_id, null);
  assert.equal(view.source?.actors[0].candidate_node_ids.length, 2);
  assert.throws(() => toEvent(event("DATA-UJI:future", "2026-10-02"), date), /future/);
});

test("DATA UJI: namespaced node/entity IDs, multi-event references, graph bounds and focus survive", () => {
  const view = toGraph(graph), events = [toEvent(event(), date), toEvent(event("DATA-UJI:ev2", "2026-09-02"), date)];
  assert.equal(view.nodes[0].id, "Deal:" + id); assert.equal(view.nodes[0].entity_id, id);
  const person = view.nodes.at(-1)!;
  assert.deepEqual(nodeEventIds(person), ["DATA-UJI:ev1", "DATA-UJI:ev2"]);
  assert.strictEqual(person.source, graph.nodes.at(-1));
  assert.equal(eventNode("DATA-UJI:ev1", view.nodes)?.id, "Event:DATA-UJI:ev1");
  assert.equal(linkedEvent(person, events, "DATA-UJI:ev1")?.event_id, "DATA-UJI:ev1");
  assert.ok(solarSystemLayout(view.nodes, events).systems.find(s => s.date === null)?.nodeIds.includes(person.id));
  assert.strictEqual(view.edges[0].provenance, graph.edges[0]); assert.strictEqual(view.meta, meta); assert.strictEqual(view.bounds, graph.bounds);
  assert.equal(view.edges[0].source, "Account:" + account); assert.equal(view.truncated, true);
  assert.throws(() => toGraph({ ...graph, nodes: [] }), /dangling/);
});

test("DATA UJI: field-level evidence becomes clickable without changing original source references", () => {
  const source = { ...event(), evidence_ids: [], status: { ...absent<string>("ambiguous"), evidence_ids: ["DATA-UJI:status"] } };
  const view = toEvent(source, date);
  assert.deepEqual(view.evidence_ids, ["DATA-UJI:status", "DATA-UJI:claim"]);
  assert.deepEqual(view.source?.evidence_ids, []); assert.equal(view.status, null);
  const raw = { ...graph, nodes: graph.nodes.map((n, i) => i === 0 ? { ...n, evidence_ids: [] } : n) };
  const mapped = toGraph(raw);
  assert.deepEqual(mapped.nodes[0].evidence_ids, ["DATA-UJI:claim"]); assert.deepEqual(mapped.nodes[0].source?.evidence_ids, []);
});

test("DATA UJI: evidence stays a scoped field/span, not a fabricated full record or Observed claim", () => {
  const view = toEvidence(evidence, date, meta);
  assert.strictEqual(view.source, evidence.evidence); assert.strictEqual(view.meta, meta);
  assert.equal(view.raw_record, undefined); assert.equal(view.information_kind, "Unknown"); assert.equal(view.verification, "ambiguous");
  assert.equal(view.title, "DATA-UJI-field"); assert.equal(view.excerpt, evidence.evidence.content_excerpt);
  assert.equal(view.source?.source_checksum, "DATA-UJI-source-checksum"); assert.equal(view.source?.record_id_kind, "derived");
  assert.deepEqual(view.source?.deal_ids, []); assert.deepEqual(view.source?.source_key, { line: "DATA-UJI-line" });
  assert.equal(toEvidence({ ...evidence, evidence: { ...evidence.evidence, information_kind: "Assessed" } }, date).information_kind, "Assessed");
  assert.throws(() => toEvidence({ ...evidence, meta: { ...meta, context_id: "DATA-UJI-other" } }, date, meta), /context mismatch/);
});

test("DATA UJI: all 50-item cursor pages are consumed and original page bounds retained", async () => {
  const calls: (string | undefined)[] = [];
  const result = await consumePages(async cursor => {
    calls.push(cursor);
    return cursor === undefined ? { meta, bounds: { truncated: true, truncation_reasons: ["DATA-UJI page limit"], next_cursor: "DATA-UJI opaque/?&=" }, values: Array.from({ length: 50 }, (_, i) => i) } :
      { meta, bounds: { truncated: true, truncation_reasons: ["DATA-UJI residual budget"], next_cursor: null }, values: [50] };
  }, date);
  assert.deepEqual(calls, [undefined, "DATA-UJI opaque/?&="]); assert.equal(result.values.length, 51);
  assert.equal(result.pages.length, 2); assert.equal(result.pages[0].bounds.next_cursor, calls[1]);
  assert.equal(result.bounds.truncated, true); assert.deepEqual(result.bounds.truncation_reasons, ["DATA-UJI residual budget"]);
});

test("DATA UJI: pagination rejects repeated cursors, changed metadata and oversized pages", async () => {
  await assert.rejects(consumePages(async () => ({ meta, bounds: { truncated: true, truncation_reasons: [], next_cursor: "DATA-UJI-loop" }, values: [] }), date), /repeated a cursor/);
  for (const changed of [{ dataset_version: "DATA-UJI-other" }, { context_id: "DATA-UJI-other" }, { as_of: "2026-09-01" }]) {
    await assert.rejects(consumePages(async cursor => ({ meta: cursor ? { ...meta, ...changed } : meta,
      bounds: cursor ? complete : { truncated: true, truncation_reasons: [], next_cursor: "DATA-UJI-next" }, values: [] }), date), /mismatch|metadata/);
  }
  await assert.rejects(consumePages(async () => ({ meta, bounds: complete, values: Array(51).fill("DATA-UJI") }), date), /50-item/);
});

test("DATA UJI: snapshot assertions reject not-ready, version/calendar and wrong-date metadata", () => {
  assert.throws(() => assertMeta({ ...meta, data_state: "not_ready" }, date), /not ready/);
  for (const changed of [{ contract_version: "DATA-UJI-other" }, { calendar_zone: "UTC" }, { context_id: "DATA-UJI-other" }, { as_of: "2026-09-01" }]) {
    assert.throws(() => assertMeta({ ...meta, ...changed }, date, meta), /mismatch|metadata/);
  }
});

test("DATA UJI: API list forwards stable date/limit and opaque cursors without access scope", async t => {
  const urls: URL[] = [];
  mockFetch(t, url => {
    urls.push(url); assert.equal(url.searchParams.get("limit"), "50"); assert.equal(url.searchParams.get("as_of"), date);
    assert.equal(url.searchParams.get("access_scope"), null);
    return Response.json({ meta, bounds: url.searchParams.has("cursor") ? complete : { truncated: true, truncation_reasons: ["DATA-UJI page"], next_cursor: "DATA-UJI opaque/?&=" },
      items: url.searchParams.has("cursor") ? [{ ...facts, deal_id: "DATA-UJI-DL005" }] : [facts] });
  });
  const result = await api.deals(date);
  assert.equal(urls.length, 2); assert.equal(urls[1].searchParams.get("cursor"), "DATA-UJI opaque/?&=");
  assert.equal(result.deals.length, 2); assert.equal(result.scope.mode, "api"); assert.equal(result.scope.meta?.data_state, "partial");
  assert.equal(result.pages?.length, 2); assert.equal(result.bounds?.truncated, false);
});

test("DATA UJI: optional JSON 503 does not blank required graph/timeline", async t => {
  const urls: URL[] = [];
  mockFetch(t, url => { urls.push(url); return coreReply(url) ?? unavailable(); });
  const context = await api.context(id, date);
  assert.equal(context.timeline.length, 1); assert.equal(context.graph.nodes.length, graph.nodes.length);
  assert.equal(context.candidates, null); assert.equal(context.candidates_state?.status, "unavailable");
  assert.equal(context.assessment, null); assert.equal(context.assessment_state?.status, "unavailable"); assert.equal(context.deal.action_count, null);
  assert.equal(urls.find(u => u.pathname.endsWith("/graph"))?.searchParams.get("depth"), "2");
  assert.ok(urls.every(u => !u.searchParams.has("access_scope")));
});

test("DATA UJI: supplied readiness zero maps without deriving other scores and retains assessment provenance", async t => {
  mockFetch(t, url => coreReply(url) ?? (url.pathname.endsWith("/assessment") ? Response.json(assessment) : unavailable()));
  const context = await api.context(id, date);
  assert.equal(context.deal.readiness, 0); assert.equal(context.deal.readiness_label, "DATA-UJI level zero");
  assert.equal(context.deal.attractiveness, null); assert.equal(context.deal.urgency, null); assert.equal(context.deal.coverage, null);
  assert.deepEqual(context.assessment, assessment); assert.equal(context.assessment?.jev_confidence, 0); assert.equal(context.assessment_state?.status, "available");
  assert.equal(applyAssessment(context.deal, { ...assessment, readiness_100: null }).readiness, null);
});

test("DATA UJI: optional 403/404 and mismatched assessment metadata remain explicit errors", async t => {
  for (const status of [403, 404]) {
    mockFetch(t, () => Response.json({ code: "DATA-UJI-denied", message: "DATA UJI resource not found" }, { status }));
    const result = await optionalRead<Assessment>("/api/data-uji", undefined, meta);
    assert.equal(result.data, null); assert.equal(result.state.status, "error"); assert.equal(result.state.http_status, status);
  }
  mockFetch(t, url => coreReply(url) ?? (url.pathname.endsWith("/assessment") ? Response.json({ ...assessment, meta: { ...meta, context_id: "DATA-UJI-mixed" } }) : unavailable()));
  const context = await api.context(id, date);
  assert.equal(context.assessment, null); assert.equal(context.assessment_state?.status, "error"); assert.equal(context.deal.readiness, null);
  assert.equal(context.graph.nodes.length, graph.nodes.length);
});

test("DATA UJI: required mixed context rejects the entire bootstrap", async t => {
  mockFetch(t, url => url.pathname.endsWith("/graph") ? Response.json({ ...graph, meta: { ...meta, dataset_version: "DATA-UJI-mixed" } }) : coreReply(url) ?? unavailable());
  await assert.rejects(api.context(id, date), /dataset mismatch/);
});

test("DATA UJI: timeline API consumes every cursor and sorts without dropping account scope", async t => {
  const urls: URL[] = [];
  mockFetch(t, url => { urls.push(url); return Response.json({ meta,
    bounds: url.searchParams.has("cursor") ? complete : { truncated: true, truncation_reasons: ["DATA-UJI page"], next_cursor: "DATA-UJI-next" },
    events: [url.searchParams.has("cursor") ? event("DATA-UJI:ev2", "2026-09-02") : event()] }); });
  const view = await api.timeline(id, date, undefined, meta);
  assert.equal(urls.length, 2); assert.equal(view.events.at(-1)?.event_id, "DATA-UJI:ev2");
  assert.equal(view.pages?.length, 2); assert.deepEqual(view.events[1].source?.deal_ids, []);
});

test("DATA UJI: Go/local errors are extracted; non-JSON/internal bodies are never shown", async t => {
  mockFetch(t, () => Response.json({ code: "data_not_ready", message: "DATA UJI safe backend message" }, { status: 503 }));
  await assert.rejects(request("/api/data-uji"), e => e instanceof ApiError && e.status === 503 && e.code === "data_not_ready" && e.message === "DATA UJI safe backend message");
  mockFetch(t, () => Response.json({ error: "DATA UJI local validation" }, { status: 400 }));
  await assert.rejects(request("/api/data-uji"), /DATA UJI local validation/);
  mockFetch(t, () => new Response("DATA UJI internal stack trace", { status: 404, headers: { "Content-Type": "text/plain" } }));
  await assert.rejects(request("/api/data-uji"), e => e instanceof ApiError && e.status === 404 && !e.message.includes("stack trace"));
  mockFetch(t, () => new Response("DATA UJI malformed JSON", { headers: { "Content-Type": "application/json" } }));
  await assert.rejects(request("/api/data-uji"), /invalid or non-JSON/);
});

test("DATA UJI: transport failure makes one API request, with no retry or CSV fallback", async t => {
  let calls = 0;
  mockFetch(t, () => { calls++; throw new Error("DATA UJI internal socket details"); });
  await assert.rejects(api.deals(date), /Backend request could not be completed/); assert.equal(calls, 1);
});

test("DATA UJI: compare/Copilot bodies stay scoped requests, never browser access declarations", async t => {
  const bodies: unknown[] = [];
  mockFetch(t, (_url, init) => { bodies.push(JSON.parse(String(init?.body))); return unavailable(); });
  await assert.rejects(api.compare(id, date, ["DATA-UJI:a", "DATA-UJI:b"]), e => e instanceof ApiError && e.status === 503);
  await assert.rejects(api.ask("DATA UJI question", date, id), e => e instanceof ApiError && e.status === 503);
  assert.deepEqual(bodies, [{ as_of: date, action_ids: ["DATA-UJI:a", "DATA-UJI:b"] }, { question: "DATA UJI question", as_of: date, deal_id: id }]);
});
