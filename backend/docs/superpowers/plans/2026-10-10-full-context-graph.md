# Full Context Graph Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use executing-plans to implement task-by-task. Execution method: Native, already selected by user.

**Goal:** Finish O1 full-source static graph preparation, scoped read-only Aura repositories and source-grounded action retrieval, with verified handoff to O2.

**Architecture:** Add a full-dataset namespace/adapter alongside the working P04 adapter. Offline preparation produces typed records, partition digest registries and backing usage rows; the one-off loader publishes readiness only after validation. Runtime queries select authorized scope/time/record types, not the complete dataset.

**Tech Stack:** Go standard library CSV/JSON/SHA256, existing Neo4j Go driver v5.28.4, existing Aura instance, PowerShell only for invocation/config export.

**Spec:** `backend/docs/superpowers/specs/2026-10-10-full-context-graph-design.md` (user approved written spec).

## Global Constraints

- Dataset FIX; business database static prepared once, no ETL pipeline/connector/scheduler/CSV startup or request.
- No internal/ or cmd/. Do not edit O2 routes/controllers/services/main.go, existing domain/interface signatures or HTTP OpenAPI.
- as_of calendar Asia/Jakarta; maximum 2026-10-01. Missing not zero/false/rejected/resolved.
- Keep P04 namespace and existing tests; no DELETE, overwrite, silent environment version switch or paid instance upgrade.
- All 15 input source hashes/counts verified; no fabricated source IDs, identities, stage history, consent, approvals or actions.
- Only source-supported fields known historically. Retrospective fields available at snapshot unless separately dated evidence supports earlier knowledge.
- Full usage contributes aggregates and stored source partitions; not 226.300 individual transaction graph nodes.
- JEV/Copilot/scoring/auth/HTTP remain O2. No claimed MVP/model success from graph acceptance.

## Review Focus

1. A shared person/employment/reply reference must not leak foreign-account facts or future evidence when only one account is authorized.
2. A decision dated earlier may contain future outcome/status_janji; filter availability per field, not just event date.
3. Daily/monthly usage must preserve offline missingness, distinct outlets and partial-month cutoff; prospects never acquire analogue usage.
4. Selected actions must fail atomically for one invalid/future/unauthorized ID; an analog approval is never current authorization.
5. Failed/retried partial load must remain unready and deterministic; selective decoder must verify partition binding without a full namespace fetch.

## Files and Interfaces

Preserve all P04 code. New production files under repository/: `full_artifact.go`, `full_validate.go`, `full_driver.go`, `full_projection.go`, `full_queries.go`, `full_repository.go`, `full_actions.go`. New one-off packages: `tools/preparefull/` and `tools/loadfull/`. Generated artifacts under `database/neo4j/full/` only after size/ignore review; large raw payloads remain local generated artifacts, not blindly committed.

Preparation transport types in `repository/full_artifact.go` are tooling contracts, not new HTTP DTOs:

- `FullArtifact{Manifest FullManifest, Partitions []FullPartition, Records []FullRecord, SourcePartitions []FullSourcePartition}`.
- `FullManifest{SchemaVersion, DatasetVersion, ManifestHash string; Sources map[string]string; SourceCounts, Counts map[string]int; DealIDs, AccountIDs []string; PartitionRoots map[string]string}`. Schema `relio-neo4j-full-v1`; ready flag is DB publication metadata, not assumed from generated JSON.
- `FullRecord{Kind, ID, PartitionID string; AccountIDs, DealIDs []string; AvailableFrom models.Date; Payload json.RawMessage; PayloadHash string}`. Kind allows deals/nodes/edges/events/evidence/templates/occurrences. Payload domain/wrapper ID must equal ID.
- `FullPartition{PartitionID, Hash string; Digests []FullDigest}`; `FullDigest{Kind, ID, Hash string}`. Partition canonical key includes source scope (account/company), availability month and kind; shared records dedup by canonical kind/ID during projection. Partition hash binds sorted kind/ID/payload hashes; manifest binds partition roots.
- `FullSourcePartition{ID, SourceFile, SourceChecksum, AccountID string; Date models.Date; Rows []map[string]string; Keys []map[string]string; Hash string}` stores native usage rows/locators. It is not a returned graph node. Bind these hashes through manifest partition roots too.
- `ValidateFullArtifact(a FullArtifact) error`, `SealFullArtifact(a *FullArtifact) error` for offline preparation/loader. No SQL/Cypher/driver exposed to services.
- `NewNeo4jFullContextRepository(ctx context.Context, c Neo4jConfig) (*Neo4jFullContextRepository,error)` and Close lifecycle; implement all five existing repository interfaces unchanged.
- Private operation/query selector and typed transport reader allow tests to exercise production projection/retrieval while substituting only network I/O. No public raw-query method or in-memory runtime fallback.

## Task 1: Source preparation and integrity artifact

Files: preparation types/validators above; `tools/preparefull/main.go`, `source.go`, `graph.go`, `usage.go`; corresponding `_test.go`; `docs/full-source-mapping.md` and `docs/full-data-quality.md`.

Consumes existing models and `docs/source-audit-snapshot.json` expected source hashes/counts. Produces FullArtifact plus source coverage/diagnostic report. CLI: `go run ./tools/preparefull -dataset ../../Datasets -out database/neo4j/full` from backend.

- [ ] Write failing tests: fixture CSV quoted comma/semicolon/blank; JSONL malformed/duplicate/native FK failures; changed source checksum refuses output. No real-source records fabricated in fixtures.
- [ ] Run `go test ./tools/preparefull ./repository -run 'TestFullSource|TestFullArtifact' -count=1`; observe missing feature RED.
- [ ] Implement standard-library parsing and source roster; validate all keys/dates/FK using actual headers. Report row-level failures safely and stop on integrity failure, never silently skip.
- [ ] Build deals/accounts/people/outlets, employment WORKED_AT, interactions/native reply and participants, decisions/contracts, support/product enrichment. Use native ID labels before snapshot when names lack dated support. Event IDs/entity IDs remain deterministic and separate.
- [ ] Create field/span evidence with source checksums; composite-source SourceRecordID empty and SourceKey native tuple. Company evidence explicitly company-scoped; cross-account links retain scope restrictions.
- [ ] Snapshot current commercial/CRM/support/roadmap values at 2026-10-01. Decision nilai/keputusan/native actors available at decision date; retrospective alasan/status_janji conservative snapshot availability. Preserve request/decision/application scope.
- [ ] Write/run tests for P04 timeline 2 then 3; P05 no interactions; C23 two native deals but no inferred interaction-deal links; K017 transition and unmatched email unresolved; history overlap preserved and end inclusive+1.
- [ ] Implement usage account/day contributions with native backing rows; monthly rollup from eligible dates only. Test sum against independent literal fixture, unique outlets across days, blank offline not zero, absent feature tuple not zero, feature month available at month end, no usage prospect attribution.
- [ ] Seal partitions/manifest, validate domain IDs/ref closure/nulls; two real-data preparation runs byte-identical. Report actual 15-source hashes/counts, graph/record/partition/source-row counts and sizes.
- [ ] Run package/full tests then commit only code/docs/tests and reviewed compact artifacts; no secret/raw dataset/cache.

## Task 2: Selective Neo4j driver and atomic readiness loader

Files: `repository/full_driver.go`, `full_queries.go`, `full_validate.go`, related tests; `tools/loadfull/main.go`, `_test.go`; full schema/manifest artifacts.

Consumes sealed FullArtifact. Produces a ready immutable full namespace and private selective transport, alongside old P04.

- [ ] Write failing tests: partition payload+adjacent hash replacement fails sealed-root check; duplicated/missing partition record/native ID mismatch rejected; foreign-scope/future data not selected; malformed target/version/manifest fails before writes.
- [ ] Run `go test ./repository ./tools/loadfull -run 'TestFullPartition|TestFullQuery|TestFullLoad' -count=1`; verify RED.
- [ ] Parameterized query predicates: namespace, kind, account/deal/company authorization, availability <= cutoff; evidence/ID lookup selected directly. Fetch only selected partition registries, verify their manifest root then payload hashes. No global record scan or complete digest inventory per request.
- [ ] Constructor validates neo4j+s/config, connectivity, explicit database, read-only sessions and Close lifecycle. Wrap driver failures in existing typed errors; never print raw Cause/config.
- [ ] One-off loader reads ignored .env safely, dry-run/preflight flags, explicit artifact path and expected version. Index scope/time/IDs and uniqueness namespace/kind/ID (and partition IDs). Calculate projected sizes and check instance capacity before live writes; paid changes require user decision.
- [ ] Batch MERGE immutable records/partitions/source partitions/adjacency into new namespace. Check existing hash before every replacement attempt; no replacement. Ready remains false until counts/digests/domain links complete; interruption resumes idempotently, no DELETE and no automatic env switch.
- [ ] Test local dry-run and failure/duplicate handling. Live write proceeds only on explicit target and adequate capacity. Verify same namespace rerun counts stable; test ready false for incomplete test-only namespace if supported safely without deleting production data.
- [ ] Run suite/vet, commit exact tooling/driver files. Do not claim live completion until acceptance Task 5.

## Task 3: Full deal/graph/timeline/evidence repositories

Files: `repository/full_repository.go`, `full_projection.go`, corresponding `_test.go`. Reuse existing snapshot/bounds/cursor/typed-error helpers; keep old P04 methods unchanged.

Consumes selective validated records + models.SnapshotContext. Implements DealFactsRepository, GraphRepository, TimelineRepository, EvidenceRepository on Neo4jFullContextRepository.

- [ ] Write failing tests: all native deal IDs distinguish account IDs; scope empty denied; missing ID vs before-created snapshot vs unready/query failure; current values historical null; evidence access/future ref closure; same ContextID across operations.
- [ ] Run focused tests RED, implement validation/projection without altering input records. Snapshot Fact masking uses existing semantics; pointers/arrays preserved, no nil-array output or numeric imputation.
- [ ] List native deals authorized by primary scope; apply deal/account/type filters, stable paging max/default 50, context/filter-bound cursor checks (cursor is not signed or auth). Do not imply paginated/subset list is ACV denominator.
- [ ] Graph primary deal/account anchors, scope-safe BFS depth max2, nodes150/edges300, retained focus under tight caps, deterministic edge/node order and truncation. Timeline includes legitimate account context and native deal events only, filters/sort/cursor stable.
- [ ] Evidence resolves exact field/span/source partition proof from DB, no whole thread/raw future records. Related cross-account refs require separate authorization; mask absent proof/unresolved identity rather than promote it.
- [ ] Tests from Review Focus1/2: K017 old/new account access split, cross-account native reply, company-only source, future ValidTo/ref, retrospective decision text and ticket resolution; no unauthorized names/candidates/evidence IDs.
- [ ] Exercise all22 deals/all45 accounts against prepared artifact with transport double; assert P01–P05 expected native mappings and P04 regression. Run full suite/vet/race, commit.

## Task 4: Source-grounded action catalog and repositories

Files: `tools/preparefull/actions.go`, tests; `repository/full_actions.go`, `_test.go`; `docs/action-rule-registry.md`; versioned machine-readable registry under `repository/testdata/` or preparation assets.

Consumes real source records, current-deal/account context, authorized analog scopes and rule version. Implements ActionRepository using existing signatures. Taxonomy/relevance versions `relio-actions-v1` and `relio-relevance-v1`; these label deterministic rules, not source-native findings.

- [ ] Audit source-supported company/Sales actions. Record every selected occurrence locator/quote/extraction/status rule; exclude buyer-only requests from fabricated completed actions. Templates reference at least one actual occurrence.
- [ ] Write failing tests: candidate without occurrence/proof rejected; no relevance yields0; one valid yields1; future/foreign precedent excluded; analog approval never copied to current deal; requested/approved/rejected/applied/offered remain distinct.
- [ ] Implement minimal reviewed deterministic taxonomy after audit (no predefined menu without source). Relevance needs visible deal/account-context gate/request evidence and precedent evidence under versioned explicit rules; no fallback all actions and no LLM inference stored as graph truth.
- [ ] Policy facts expose integer BPS only from source-supported value; unknown approver role/consent/outcome remain nullable. Outcome future/current retrospective field masked. Native decision/application IDs not joined by amount/date resemblance.
- [ ] ReadActionCandidates pages stable max50 with actual TotalValidCount and ContextID; caller mismatch/unknown rule version InvalidQuery/ContextMismatch as appropriate. ReadSelectedActions requires 1–4 unique IDs (O2 comparison2–4), validates complete same-snapshot candidate set and returns complete dedup evidence or InvalidSelection.
- [ ] Add tests for duplicate/one bad/future/unauthorized selection atomic rejection and cursor/date mismatch. Bundle cap, if configured, is an explicit opt-in limit and atomic BundleLimitExceeded; default not falsely declared agreed with O2.
- [ ] Measure real 1–4 valid-option bundles where available; otherwise report actual insufficiency, never pad. Generate source-grounded evaluation facts/cases for graph/action/policy; not model-produced ground truth.
- [ ] Run suite/vet/race and commit exact implementation/registry/tests/docs.

## Task 5: Live acceptance, O2 handoff and branch publication

Files: `repository/full_integration_test.go`, `docs/full-context-runbook.md`, `docs/full-context-o2-handoff.md`, `evaluation/o1-source-cases.md`; updates to this plan/status only after verified output.

- [ ] Add opt-in read-only `TestNeo4jFullLive`, default SKIP; enabled missing config FAIL. Separate full version opt-in from P04 so old live suite doesn't silently query wrong namespace. No test imports/seed writes.
- [ ] Confirm schema/full manifest counts/roots/readiness on Aura, then run five demo deals at historical/current dates; graph/timeline/evidence closure; scoped analog actions/selected bundles; support/usage snapshots and no-future-leakage. Capture actual counts/bytes/latencies, not asserted target p95.
- [ ] `go test ./... -count=1`, `go vet ./...`, `go test -race ./... -count=1`, opt-in full live test; prepared files twice identical; formatting/diff/secret/ignore checks.
- [ ] One fresh whole-branch reviewer if tool available; one test-first fix pass for important/critical findings. Record unresolved minor/consumer-contract questions and limitations, not silently waive.
- [ ] Handoff existing five interfaces, new constructor/lifecycle/config/version, registry versions, exact passing examples, source coverage, safe errors, evidence budget measurements and O2 integration tasks. JEV/chatbot not claimed implemented; shared cap/frozen DTO require O2 ack.
- [ ] Commit and push feature branch under user-authorized workflow, preserve latest main changes safely; do not merge main or publish a new PR without explicit request. Keep .env, dataset and cache out of Git.

## Definition of completion and stop conditions

O1 complete only when whole-source preparation/integrity, five repository interfaces, source-backed actions, full Aura acceptance and handoff are verified. A capacity problem/secret issue requires specific user action, not a fabricated complete state. Finish safe local tasks even if live load is blocked. Irreversible deletion, paid upgrade, commercial action and O2 interface migration are outside authority.

Plan self-review: each source group, access/temporal/action rule and Review Focus is assigned to a testable task; signatures unchanged; raw daily transactions are source backing, not graph nodes. Spec wins if a plan assumption conflicts. Implementation has not started for this plan.
