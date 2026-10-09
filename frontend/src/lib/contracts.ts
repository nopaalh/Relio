// Canonical read DTOs mirror backend/models. Access scope stays server-owned.
export type FactState = "known" | "unknown" | "ambiguous" | "snapshot_only";
export type TemporalBasis = "event" | "interval" | "snapshot" | "undated";
export type Fact<T> = { value: T | null; state: FactState; temporal_basis: TemporalBasis; evidence_ids: string[]; limitations: string[] };
export type Unknown = { path: string; reason: string; reference_ids: string[] };
export type SnapshotMeta = {
  contract_version: string; dataset_version: string; as_of: string; max_as_of: string;
  calendar_zone: string; context_id: string; data_state: "ready" | "partial" | "not_ready";
  limitations: string[]; unknowns: Unknown[];
};
export type Bounds = { truncated: boolean; truncation_reasons: string[]; next_cursor: string | null };
export type PageInfo = { meta: SnapshotMeta; bounds: Bounds };
export type DealFacts = {
  deal_id: string; account_id: string; deal_type: Fact<string>; owner_id: Fact<string>;
  stage: Fact<string>; stage_since: Fact<string>; created_at: Fact<string>;
  planned_outlets: Fact<number>; potential_acv_idr: Fact<number>; status: Fact<string>;
  record_evidence_ids: string[];
};
export type ParticipantRef = {
  node_id: string | null; raw_ref: string | null; verification_state: string;
  match_method: string; candidate_node_ids: string[]; evidence_ids: string[];
};
export type Validity = { valid_from: string | null; valid_to: string | null; open_end_reason: string | null; temporal_basis: TemporalBasis };
export type CanonicalNode = {
  node_id: string; node_type: string; entity_id: string; person_kind: string | null;
  person: ParticipantRef | null; label: Fact<string>; validity: Validity;
  verification_state: string; evidence_ids: string[]; event_ids: string[];
};
export type CanonicalEdge = {
  edge_id: string; edge_type: string; source: string; target: string; validity: Validity;
  source_timestamp: string | null; observed_at: string | null; recorded_at: string | null;
  verification_state: string; match_method: string; evidence_ids: string[]; event_ids: string[];
};
export type CanonicalEvent = {
  event_id: string; verification_state: string; event_at: string; time_precision: string;
  source_timestamp: string | null; event_type: string; status: Fact<string>; raw_status: string | null;
  summary: Fact<string>; actors: ParticipantRef[]; targets: ParticipantRef[]; participants: ParticipantRef[];
  scope_kind: string; deal_ids: string[]; account_ids: string[]; node_ids: string[]; edge_ids: string[]; evidence_ids: string[];
};
export type SourceSpan = { start: number; end: number; field_checksum: string };
export type CanonicalEvidence = {
  evidence_id: string; source_file: string; source_record_id: string; record_id_kind: "native" | "derived";
  source_key: Record<string, string>; source_checksum: string; source_field: string; span: SourceSpan | null;
  source_date: string | null; source_timestamp: string | null; temporal_basis: TemporalBasis;
  content_excerpt: string; scope_kind: string; verification_state: string;
  deal_ids: string[]; account_ids: string[]; event_ids: string[]; edge_ids: string[]; occurrence_ids: string[];
  information_kind?: InformationKind;
};
export type DealPage = PageInfo & { items: DealFacts[] };
export type DealResult = { meta: SnapshotMeta; deal: DealFacts };
export type GraphResult = PageInfo & { nodes: CanonicalNode[]; edges: CanonicalEdge[] };
export type TimelineResult = PageInfo & { events: CanonicalEvent[] };
export type EvidenceResult = { meta: SnapshotMeta; evidence: CanonicalEvidence };

// View models keep the canonical source beside presentation-only fields.
export type Verification = string;
export type InformationKind = "Observed" | "Derived" | "Assessed" | "Unknown";
export type Scope = {
  as_of: string; snapshot_date: string; evidence_version: string; mode: "local" | "api";
  limitations: string[]; meta?: SnapshotMeta; unknowns?: Unknown[];
};
export type Deal = {
  deal_id: string; account_id: string; name: string; industry: string | null; city: string | null;
  owner: string; owner_id: string | null; stage: string | null; stage_since: string | null;
  acv: number | null; outlets: number | null; attractiveness: number | null;
  urgency: number | null; readiness: number | null; readiness_label: string | null;
  coverage: number | null; coverage_dimensions: { key: string; label: string; present: boolean; evidence_ids: string[] }[];
  blocker: string | null; blocker_evidence_ids: string[]; action_count: number | null;
  temporal_scope: "snapshot_only" | "dated"; assessment_status: string;
  record_evidence_ids?: string[]; facts?: DealFacts; meta?: SnapshotMeta; unknowns?: Unknown[];
};
export type TimelineEvent = {
  event_id: string; interaction_id: string | null; date: string; title: string;
  type: string; actor: string; actor_email: string | null; status: string | null;
  excerpt: string | null; evidence_ids: string[]; verification: Verification; source?: CanonicalEvent;
};
export type GraphNode = {
  id: string; label: string; type: string; subtitle: string; verification: Verification;
  evidence_ids: string[]; event_id?: string; event_ids?: string[]; entity_id?: string;
  x?: number; y?: number; source?: CanonicalNode;
};
export type GraphEdge = {
  id: string; source: string; target: string; type: string; verification: Verification;
  evidence_ids: string[]; event_ids?: string[]; provenance?: CanonicalEdge;
};
export type Evidence = {
  evidence_id: string; source_record_id: string; source_file: string; date: string | null;
  title: string; excerpt: string | null; raw_record?: Record<string,string>; information_kind: InformationKind;
  account_id: string | null; verification: Verification; source?: CanonicalEvidence; meta?: SnapshotMeta;
};
export type ActionCandidate = {
  action_id: string; title: string; description: string; eligibility: "requires_validation";
  occurrences: { occurrence_id: string; account_id: string; account_name: string; date: string;
    status: string; outcome_observed: string | null; evidence_ids: string[] }[];
  policy_flags: string[]; prerequisites: string[]; unknowns: string[];
};
export type ComparisonItem = {
  action_id: string; suitability_score_100: number | null; jev_raw_score: number | null;
  jev_confidence: number | null; jev_choice_preference: number | null; rank: number | null;
  evidence_ids: string[]; precedent_ids: string[]; policy_flags: string[]; unknowns: string[];
};
export type Comparison = {
  deal_id: string; as_of: string; comparison_type: string; assessment_status: string;
  rubric_version: string; items: ComparisonItem[]; warning: string;
};
export type FeatureState = { status: "available" | "unavailable" | "error"; message?: string; code?: string; http_status?: number };
export type Assessment = {
  meta?: SnapshotMeta; status: string; rubric_version: string; explanation: string; unknowns: string[];
  readiness_100?: number | null; readiness_label?: string | null; jev_raw_score?: number | null;
  jev_confidence?: number | null; evidence_ids?: string[]; provider_model?: string | null;
};
export type TimelineView = {
  events: TimelineEvent[]; navigation?: { previous: string | null; next: string | null };
  meta?: SnapshotMeta; bounds?: Bounds; pages?: PageInfo[];
};
export type ContextBundle = {
  catalogue_version?: string; navigation?: TimelineView["navigation"]; scope: Scope; deal: Deal;
  timeline: TimelineEvent[]; timeline_meta?: SnapshotMeta; timeline_bounds?: Bounds; timeline_pages?: PageInfo[];
  graph: { nodes: GraphNode[]; edges: GraphEdge[]; unknowns: string[]; truncated: boolean; meta?: SnapshotMeta; bounds?: Bounds };
  candidates: ActionCandidate[] | null; candidates_state?: FeatureState;
  assessment: Assessment | null; assessment_state?: FeatureState;
};
export type DealsResponse = { scope: Scope; deals: Deal[]; bounds?: Bounds; pages?: PageInfo[] };
export type CopilotResponse = {
  answer: string; status: string; evidence_ids: string[]; unknowns: string[];
  as_of: string; deal_id: string | null;
};
