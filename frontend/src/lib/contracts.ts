export type Verification = "verified" | "inferred" | "ambiguous";
export type InformationKind = "Observed" | "Derived" | "Assessed" | "Unknown";
export type Scope = { as_of: string; snapshot_date: string; evidence_version: string; mode: "local" | "api"; limitations: string[] };
export type Deal = {
  deal_id: string; account_id: string; name: string; industry: string; city: string;
  owner: string; owner_id: string; stage: string | null; stage_since: string | null;
  acv: number | null; outlets: number | null; attractiveness: number | null;
  urgency: number | null; readiness: number | null; readiness_label: string | null;
  coverage: number; coverage_dimensions: { key: string; label: string; present: boolean; evidence_ids: string[] }[];
  blocker: string | null; blocker_evidence_ids: string[]; action_count: number;
  temporal_scope: "snapshot_only" | "dated"; assessment_status: string;
};
export type TimelineEvent = {
  event_id: string; interaction_id: string; date: string; title: string;
  type: string; actor: string; actor_email: string; status: string;
  excerpt: string; evidence_ids: string[]; verification: Verification;
};
export type GraphNode = {
  id: string; label: string; type: string; subtitle: string; verification: Verification;
  evidence_ids: string[]; event_id?: string; x: number; y: number;
};
export type GraphEdge = {
  id: string; source: string; target: string; type: string; verification: Verification; evidence_ids: string[];
};
export type Evidence = {
  evidence_id: string; source_record_id: string; source_file: string; date: string | null;
  title: string; excerpt: string; raw_record: Record<string,string>; information_kind: InformationKind;
  account_id: string; verification: Verification;
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
export type ContextBundle = { catalogue_version?: string; navigation?: { previous: string | null; next: string | null }; scope: Scope; deal: Deal; timeline: TimelineEvent[]; graph: {
  nodes: GraphNode[]; edges: GraphEdge[]; unknowns: string[]; truncated: boolean;
}; candidates: ActionCandidate[]; assessment: { status: string; rubric_version: string; explanation: string; unknowns: string[] } };
export type DealsResponse = { scope: Scope; deals: Deal[] };
export type CopilotResponse = {
  answer: string; status: string; evidence_ids: string[]; unknowns: string[];
  as_of: string; deal_id: string | null;
};
