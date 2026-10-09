# Full-context handoff to O2

Status (2026-10-10): O1 full preparation, five repository interfaces and Aura acceptance verified. API/JEV/Copilot integration remains O2. Main c44620e includes commit810b7bf deleting backend/; no merge/restoration performed. Feature branch remains isolated pending user direction on that architecture change.

## Wiring (O2 responsibility)

Use NewNeo4jFullContextRepository(ctx, Neo4jConfig) and defer Close at application shutdown. It implements existing DealFactsRepository, GraphRepository, TimelineRepository, EvidenceRepository and ActionRepository without driver/query types leaking to services. Legacy DealRepository is intentionally unchanged and not bridged through lossy Deal conversion. Existing P04 adapter/namespace/config remain intact.

Construct SnapshotContext with NewSnapshotContext(asOf, datasetVersion, authorizedAccess). Max2026-10-01, Asia/Jakarta date. Use the full manifest dataset version explicitly; changing active backend configuration is O2 coordination, not automatic import behavior. Accounts/deals/analog access/company permission are supplied by trusted server authorization, not browser input.

All operations share ContextID. Snapshot fields historical=null/snapshot_only, not zero. Deal IDs are native DL-001.., not account IDs P01/C01. Historical snapshots before native creation return NotVisible. Account interactions remain account context, not guessed native deal links. Timeline has contact transition/support/contract events as appropriate; P04 interaction count2 on9/1 then3 on9/22, not necessarily total timeline count.

Graph: bounded deterministic BFS, depth<=2, nodes<=150, edges<=300, explicit truncation, focus retained under tight caps. List/timeline/action paging default/max50 and context/filter-bound cursors. Cursor is not authorization. Commercial scoring/ACV denominator must not use a paginated subset as the complete portfolio.

Evidence: exact field units, native source ID or composite key without invented SourceRecordID, source SHA, availability filters, no entire future-containing thread. Usage evidence ID ev:usage:{account_id}|{YYYY-MM} resolves stored monthly backing filtered to as_of, including partial month. JSON excerpt preserves missing offline count and distinct outlets; not a synthesized native source record. Graph usage nodes represent completed months; partial month is explicitly resolved as evidence, not mislabeled as completed rollup.

Unassigned external-contact employment (native account_id blank) stays stored but is not served under AllowCompanyEvidence; that flag is not authorization for foreign personal history. Such enrichment needs an explicit agreed permission contract, not a guessed current-account association. Future focus event ID lookup returns NotVisible; unauthorized focus returns AccessDenied without returning future/foreign fields.

Actions: explicit relio-relevance-v1; source taxonomy relio-actions-v1. ReadActionCandidates returns actual count, including0/1. ReadSelectedActions accepts1–4 unique valid IDs and fails atomically, with full deduplicated proof fields; O2 enforces2–4 comparison. Relation current_deal/account_context/analog_precedent must be retained. Policy facts belong to each occurrence, never copied as new-request permission. Keyword relevance is historical-context relevance, not unresolved-gate/suitability inference. No reference consent is fabricated. Registry: action-rule-registry.md.

MaxSelectedEvidenceBytes defaults0, configurable opt-in atomic cap. Shared final limit/DTO acknowledgment remains O2 decision. Wire safe error mapping for NotFound, NotVisible, DataNotReady, AmbiguousReference, QueryFailed, AccessDenied, ContextMismatch, InvalidQuery, InvalidSelection and BundleLimitExceeded; do not return Cause/entity IDs blindly.

## What remains outside O1

HTTP routes/controllers/DTOs, service orchestration/scoring, JEV adapter/action comparison, model cache, Copilot/LLM, auth/access construction and frontend. No two-server architecture required. O2 should consume these repositories, not duplicate source mapping or graph queries. No commercial automation/new discount approval.

## Verified publication / acceptance

Version: `full:d8e235c9a52b20fc6a16711f292744f3a06ff52096297ca3805e9f99dcb05a40`. Schema relio-neo4j-full-v1. Two CLI preparations byte-identical;15sources /229627rows;22deals /45accounts;4537domain nodes /7535edges /1284events /19811evidence;24occurrences /12templates;480monthly backing containers retain226300daily rows. Artifact80,921,871bytes;33,225typed records /2,914sealed partitions.

Aura full namespace ready; loader rerun idempotent with exact counts unchanged. Full namespace36,666physical nodes /40,857relationships (includes typed records, digest registries, scopes and backing). Whole database36,713nodes /40,865relationships, including47old P04 nodes /8relationships. No DELETE, paid upgrade or .env switch.

Local prepared acceptance covered all22deals and all timeline field proofs for five prospect scopes at9/1,9/22,10/1, including not-yet-created P03/P05. Latest read-only full Aura acceptance PASS56.14s: five prospect/date combinations, graph/timeline/deal/context consistency, one representative field proof per visible event, all selected-action bundle proof fields, partial C01 September usage, denied cross-account/unscoped employment evidence and future focus NotVisible. P04 interaction count2→3 confirmed. Read-only legacy P04 acceptance PASS33.52s after full publication.

P02/P04 have3 source-backed discount templates under authorized analog C01/C23/C24; other demo contexts0. Three-option live bundles26,534–27,426bytes. Combined per-deal sequence(find/timeline/representative evidence/graph/candidates/selected)2,369–5,434ms in the latest acceptance run; not per-query latency, benchmark or p95. These candidates are historical price-context relevance, not reference suitability, unresolved-gate proof or new-request approval.

go test ./... and go vet ./... PASS; full go test -race ./... PASS. New checksum/FK acceptance tests also PASS; they mutate temporary copies/in-memory data only. Fresh read-only reviewer found one Important query property mismatch; regression RED→GREEN and live partial usage PASS. Decoder-error and missing-known-container findings were regraded Important and fixed test-first. Remaining minor: repeated locator-redaction Meta limitation can be deduplicated. No HTTP/frontend/JEV/LLM runtime claim.

Machine-readable review registry: repository/testdata/full-action-rules.json; executable source of truth remains O1 extraction/query code and versioned manifest. O2 must not implement a second mapping from this review metadata.
