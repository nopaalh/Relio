import type {
  Bounds, CanonicalEvent, ContextBundle, Deal, DealFacts, DealResult, Evidence, EvidenceResult,
  Fact, GraphResult, PageInfo, Scope, SnapshotMeta, TimelineEvent, Unknown,
} from "./contracts.ts";
import { validDate } from "./domain.ts";

const strings = (value: unknown): value is string[] => Array.isArray(value) && value.every(v => typeof v === "string");
const nonblank = (value: unknown): value is string => typeof value === "string" && Boolean(value.trim());

export function assertMeta(meta: SnapshotMeta, date: string, expected?: SnapshotMeta): void {
  if (!meta || ![meta.contract_version, meta.dataset_version, meta.context_id].every(nonblank) ||
      meta.as_of !== date || !validDate(meta.as_of) || !validDate(meta.max_as_of) || meta.as_of > meta.max_as_of ||
      meta.calendar_zone !== "Asia/Jakarta" || !["ready", "partial", "not_ready"].includes(meta.data_state) ||
      !strings(meta.limitations) || !Array.isArray(meta.unknowns) ||
      meta.unknowns.some(u => !u || !nonblank(u.path) || !nonblank(u.reason) || !strings(u.reference_ids))) {
    throw new Error("Invalid backend snapshot metadata. Refresh before combining context panels.");
  }
  if (meta.data_state === "not_ready") throw new Error("Backend data is not ready; no successful empty view can be inferred.");
  if (expected) {
    assertDataset(meta, expected);
    if (meta.as_of !== expected.as_of || meta.context_id !== expected.context_id) {
      throw new Error("Backend context mismatch. Refresh to avoid mixing snapshots or access contexts.");
    }
  }
}

// A calendar index intentionally uses the latest date, but must use the same dataset.
export function assertDataset(meta: SnapshotMeta, expected: SnapshotMeta): void {
  if (meta.dataset_version !== expected.dataset_version || meta.contract_version !== expected.contract_version ||
      meta.calendar_zone !== expected.calendar_zone || meta.max_as_of !== expected.max_as_of) {
    throw new Error("Backend dataset mismatch. Refresh to avoid mixing dataset or contract versions.");
  }
}

export function assertBounds(bounds: Bounds): void {
  if (!bounds || typeof bounds.truncated !== "boolean" || !strings(bounds.truncation_reasons) ||
      (bounds.next_cursor !== null && !nonblank(bounds.next_cursor)) || (bounds.next_cursor !== null && !bounds.truncated)) {
    throw new Error("Invalid backend pagination bounds.");
  }
}

export function knownValue<T>(fact: Fact<T>): T | null {
  if (!fact || !["known", "unknown", "ambiguous", "snapshot_only"].includes(fact.state) ||
      !["event", "interval", "snapshot", "undated"].includes(fact.temporal_basis) ||
      !strings(fact.evidence_ids) || !strings(fact.limitations)) throw new Error("Invalid backend fact.");
  if (fact.state !== "known") {
    if (fact.value !== null) throw new Error("Unresolved backend fact must not select a value.");
    return null;
  }
  if (fact.value === null || !fact.evidence_ids.length ||
      (typeof fact.value !== "string" && typeof fact.value !== "number") ||
      (typeof fact.value === "string" && !fact.value.trim()) ||
      (typeof fact.value === "number" && !Number.isSafeInteger(fact.value))) {
    throw new Error("Known backend fact requires a supported value and provenance.");
  }
  return fact.value;
}

export type ReadPage<T> = PageInfo & { values: T[] };
export async function consumePages<T>(load: (cursor?: string) => Promise<ReadPage<T>>, date: string, expected?: SnapshotMeta) {
  const values: T[] = [], pages: PageInfo[] = [], seen = new Set<string>();
  let cursor: string | undefined, meta = expected;
  for (;;) {
    const page = await load(cursor);
    assertMeta(page.meta, date, meta);
    assertBounds(page.bounds);
    if (!Array.isArray(page.values) || page.values.length > 50) throw new Error("Backend page exceeds the 50-item contract.");
    meta ??= page.meta;
    pages.push({ meta: page.meta, bounds: page.bounds });
    values.push(...page.values);
    const next = page.bounds.next_cursor;
    if (next === null) return { meta: pages[0].meta, bounds: page.bounds, values, pages };
    if (seen.has(next)) throw new Error("Backend repeated a cursor. Pagination stopped to avoid a loop.");
    if (pages.length >= 100) throw new Error("Backend pagination exceeded the 100-page UI budget; results are not complete.");
    seen.add(next);
    cursor = next;
  }
}

export function toScope(meta: SnapshotMeta, other: SnapshotMeta[] = []): Scope {
  const metas = [meta, ...other];
  return {
    as_of: meta.as_of, snapshot_date: meta.max_as_of, evidence_version: meta.dataset_version, mode: "api", meta,
    limitations: [...new Set(metas.flatMap(m => m.limitations))], unknowns: metas.flatMap(m => m.unknowns),
  };
}

function unresolved(path: string, fact: Fact<string> | Fact<number>): Unknown[] {
  return fact.state === "known" ? [] : [{ path, reason: fact.limitations.join(" ") || `Value is ${fact.state}; not a known fact.`, reference_ids: fact.evidence_ids }];
}

export function toDeal(facts: DealFacts, meta: SnapshotMeta): Deal {
  if (!facts || !nonblank(facts.deal_id) || !nonblank(facts.account_id) || !strings(facts.record_evidence_ids)) throw new Error("Invalid backend deal facts.");
  const scalarFacts = Object.entries(facts).filter((entry): entry is [string, Fact<string> | Fact<number>] =>
    typeof entry[1] === "object" && entry[1] !== null && "state" in entry[1]);
  scalarFacts.forEach(([, fact]) => knownValue<string | number>(fact));
  const owner = knownValue(facts.owner_id);
  const missing: Unknown[] = [
    { path: "name", reason: "Company name is not supplied; the account ID is shown instead.", reference_ids: [] },
    { path: "attractiveness", reason: "Attractiveness is unavailable: this scoped list is not a full-population ACV denominator.", reference_ids: [] },
    { path: "urgency", reason: "Urgency assessment is not supplied by the backend.", reference_ids: [] },
    { path: "readiness", reason: "Readiness remains unassessed until the assessment service supplies a result.", reference_ids: [] },
    { path: "coverage", reason: "Five-dimension evidence coverage is not supplied; partial context is not a coverage score.", reference_ids: [] },
    { path: "action_count", reason: "Action catalogue is unavailable; its count is unknown, not zero.", reference_ids: [] },
  ];
  return {
    deal_id: facts.deal_id, account_id: facts.account_id, name: facts.account_id, industry: null, city: null,
    owner: owner ?? "Owner unknown", owner_id: owner, stage: knownValue(facts.stage), stage_since: knownValue(facts.stage_since),
    acv: knownValue(facts.potential_acv_idr), outlets: knownValue(facts.planned_outlets),
    attractiveness: null, urgency: null, readiness: null, readiness_label: null, coverage: null, coverage_dimensions: [],
    blocker: null, blocker_evidence_ids: [], action_count: null, temporal_scope: "snapshot_only", assessment_status: "not_assessed",
    record_evidence_ids: facts.record_evidence_ids, facts, meta,
    unknowns: [...meta.unknowns, ...scalarFacts.flatMap(([key, fact]) => unresolved(key, fact)), ...missing],
  };
}

export function dealEvidenceIds(deal: Deal): string[] {
  const factIds = deal.facts ? Object.values(deal.facts).flatMap(value =>
    value && typeof value === "object" && "evidence_ids" in value ? value.evidence_ids : []) : [];
  return [...new Set([...(deal.record_evidence_ids ?? []), ...factIds, ...deal.blocker_evidence_ids])];
}

export function toEvent(event: CanonicalEvent, date: string): TimelineEvent {
  if (!event || !nonblank(event.event_id) || !validDate(event.event_at) || event.event_at > date ||
      !nonblank(event.event_type) || !strings(event.evidence_ids) || !strings(event.deal_ids) || !strings(event.account_ids) ||
      !Array.isArray(event.actors) || !Array.isArray(event.targets) || !Array.isArray(event.participants)) throw new Error("Invalid or future backend timeline event.");
  const actor = event.actors.map(ref => {
    if (ref.verification_state === "verified" && ref.node_id) return ref.raw_ref ?? ref.node_id;
    return ref.raw_ref ? `${ref.raw_ref} (${ref.verification_state || "unresolved"})` : "Actor unknown";
  }).join(" · ") || "Actor unknown";
  return {
    event_id: event.event_id, interaction_id: null, date: event.event_at, title: event.event_type, type: event.event_type,
    actor, actor_email: null, status: knownValue(event.status), excerpt: knownValue(event.summary),
    evidence_ids: [...new Set([...event.evidence_ids, ...event.status.evidence_ids, ...event.summary.evidence_ids])], verification: event.verification_state, source: event,
  };
}

export function toGraph(result: GraphResult): ContextBundle["graph"] {
  assertBounds(result.bounds);
  if (!Array.isArray(result.nodes) || !Array.isArray(result.edges)) throw new Error("Invalid backend graph.");
  const ids = new Set(result.nodes.map(n => n.node_id));
  if (ids.size !== result.nodes.length || new Set(result.edges.map(e => e.edge_id)).size !== result.edges.length) throw new Error("Duplicate backend graph IDs.");
  return {
    meta: result.meta, bounds: result.bounds, truncated: result.bounds.truncated,
    unknowns: result.meta.unknowns.map(u => `${u.path}: ${u.reason}`),
    nodes: result.nodes.map(n => {
      if (!nonblank(n.node_id) || !nonblank(n.entity_id) || !nonblank(n.node_type) || !strings(n.event_ids) || !strings(n.evidence_ids)) throw new Error("Invalid backend graph node.");
      const label = knownValue(n.label);
      return { id: n.node_id, entity_id: n.entity_id, type: n.node_type, label: label ?? n.entity_id,
        subtitle: label === null ? `Label ${n.label.state} · ${n.entity_id}` : n.entity_id,
        verification: n.verification_state, evidence_ids: [...new Set([...n.evidence_ids, ...n.label.evidence_ids])], event_ids: n.event_ids, source: n };
    }),
    edges: result.edges.map(e => {
      if (!nonblank(e.edge_id) || !nonblank(e.edge_type) || !ids.has(e.source) || !ids.has(e.target) ||
          !strings(e.evidence_ids) || !strings(e.event_ids)) throw new Error("Invalid backend graph edge or dangling endpoint.");
      return { id: e.edge_id, source: e.source, target: e.target, type: e.edge_type, verification: e.verification_state,
        evidence_ids: e.evidence_ids, event_ids: e.event_ids, provenance: e };
    }),
  };
}

export function toEvidence(result: EvidenceResult, date: string, expected?: SnapshotMeta): Evidence {
  assertMeta(result.meta, date, expected);
  const e = result.evidence;
  if (!e || ![e.evidence_id, e.source_record_id, e.source_file, e.source_field, e.source_checksum].every(nonblank) ||
      typeof e.content_excerpt !== "string" || !strings(e.account_ids) || !strings(e.deal_ids) ||
            !strings(e.event_ids) || !strings(e.edge_ids) || !strings(e.occurrence_ids) ||
            !e.source_key || typeof e.source_key !== "object" || Array.isArray(e.source_key) || Object.values(e.source_key).some(v => typeof v !== "string") ||
      (e.source_date !== null && (!validDate(e.source_date) || e.source_date > date)) ||
            (e.span !== null && (!e.span || !Number.isSafeInteger(e.span.start) || !Number.isSafeInteger(e.span.end) || e.span.start < 0 || e.span.end < e.span.start || !nonblank(e.span.field_checksum)))) throw new Error("Invalid or future backend evidence.");
  return {
    evidence_id: e.evidence_id, source_record_id: e.source_record_id, source_file: e.source_file,
    title: e.source_field, date: e.source_date, excerpt: e.content_excerpt || null,
    information_kind: e.information_kind ?? "Unknown", account_id: e.account_ids.length === 1 ? e.account_ids[0] : null,
    verification: e.verification_state, source: e, meta: result.meta,
  };
}

export function applyAssessment(deal: Deal, assessment: ContextBundle["assessment"]): Deal {
  if (!assessment) return deal;
  const readiness = assessment.readiness_100 ?? null;
  if (readiness !== null && (!Number.isFinite(readiness) || readiness < 0 || readiness > 100)) throw new Error("Invalid backend readiness score.");
  return { ...deal, readiness, readiness_label: readiness === null ? null : assessment.readiness_label ?? null,
    assessment_status: assessment.status, unknowns: readiness === null ? deal.unknowns : deal.unknowns?.filter(u => u.path !== "readiness") };
}

export function assertDealResult(result: DealResult, id: string, date: string): void {
  assertMeta(result.meta, date);
  if (result.deal?.deal_id !== id) throw new Error("Backend returned a different deal than requested.");
}
