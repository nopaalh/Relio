# Neo4j P04 Context Slice Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [x]`) syntax for tracking.

**Goal:** Siapkan seed Cypher nyata DL-004/P04 dan adapter Go read-only untuk deal facts, graph, timeline dan evidence.

**Architecture:** Preparation offline PowerShell menghasilkan artefak statis yang dimuat manual sekali ke Aura. Adapter di repository/ memakai managed read transaction; proyeksi domain dipisah dari transport driver sehingga temporal/scope/reference tests tidak memerlukan instance. Runtime tidak membaca CSV/JSON lokal atau seed sebagai fallback.

**Tech Stack:** Go floor 1.22, Neo4j Go driver v5.28.4 (official go.mod floor 1.18), PowerShell, Cypher constraints/MERGE. Tidak membutuhkan APOC atau web-hosted CSV.

**Spec:** [desain slice](../specs/2026-10-10-neo4j-context-slice-design.md), disetujui user dengan "gas nomor 1" pada 10 Oktober 2026. Deklarasi consumer tetap candidate untuk O2; tidak mengubah interface/public DTO.

## Global Constraints

- Dataset FIX; database bisnis STATIS; hanya preparation manual sekali, tanpa ETL pipeline/connector/scheduler/refresh atau parsing CSV pada startup/request.
- as_of maksimal 2026-10-01, date-only seluruh hari Asia/Jakarta; interval internal half-open.
- Account-only interaction tetap account-scoped; meeting native IDs berbeda dari unresolved historical email.
- Unknown/null tidak menjadi zero/false; tidak invent recorded_at, stage history, approval, consent, action atau outcome.
- Tidak mengubah main/routes/controllers/services/models existing/interface/OpenAPI/arsitektur bersama; tidak membuat internal/ atau cmd/.
- Write set baru tools/, backend/database/neo4j/, backend/repository/neo4j_*.go dan docs runbook; hanya go.mod/go.sum existing berubah untuk driver yang dipin. Tidak bump Go floor.
- Tidak load database, publish, commit/push/merge otomatis. Live Aura acceptance menunggu target instance dan environment secrets server yang diberikan aman.
- Pilih branch implementasi terpisah dari PR #4, dari HEAD yang memuat shared declarations; tidak overwrite user changes. Branch lokal feat/neo4j-context-o1 disarankan, tanpa worktree tambahan jika user mempertahankan native checkout.

## Review Focus

- Source scalar/quote berisi Unicode, apostrophe, backslash atau newline: JSON/Cypher escaping tidak mengubah isi; checksum berasal dari bytes asli.
- Future records, snapshot facts, employment end, nested participant/evidence/event refs: tidak bocor melalui metadata atau refs walau scalar utama sudah dimask.
- Repeated reads/different as_of/concurrent requests: projection tidak memutasi stored input atau mengubah hasil request berikutnya.
- Namespace manifest mismatch/incomplete seed/partial coverage: data_not_ready, bukan empty success atau overwrite namespace.
- Focus/filter/cursor/scope changes: tidak memperluas akses dan cursor replay pada context/filter lain ditolak.

## Shared storage shape

Private structs di neo4j_records.go: storedContext { Manifest storedManifest; Deals []storedDeal; Nodes []storedNode; Edges []storedEdge; Events []models.Event; Evidence []storedEvidence }. Wrapper storedDeal/Node/Edge/Evidence memiliki Value domain type dan AvailableFrom models.Date. storedManifest memiliki DatasetVersion, SchemaVersion, ManifestHash, Ready, DealIDs, AccountIDs, source file hashes dan expected record counts. Gunakan JSON snake_case eksplisit dan initialized collections.

Artifact context-p04.json memakai shape tersebut untuk preparation/tests saja. Seed menyimpan per-record domain JSON payload + available_from + scope/version/ID sebagai properties; Neo4j menyimpan adjacency graph nyata, bukan satu blob response. Events disimpan date-only, scope IDs dan participant references. Driver mengembalikan records dari database ke storedContext, tidak membaca context-p04.json.

visibleContext di neo4j_projection.go memiliki Deals/Nodes/Edges/Events/Evidence serta snapshot Meta. projectContext(raw storedContext, snapshot models.SnapshotContext) (visibleContext, error) tidak melakukan I/O dan tidak memutasi raw. visibleContext berisi hanya authorized/date-visible refs; response methods menambahkan bounds/filter sesuai kebutuhan.

## Task 1 — Preparation source-grounded dan schema/seed statis

**Files:** Create tools/prepare-neo4j-p04.ps1; backend/database/neo4j/schema.cypher, seed-p04.cypher, context-p04.json, manifest-p04.json, .gitattributes; backend/repository/neo4j_records.go, neo4j_seed_test.go.

**Interfaces:** Script parameters -DatasetPath (required) dan -OutputDirectory (required); input expected hashes dari handoff source-cases-draft.json, dibaca offline. Output shape Shared storage shape. SchemaVersion relio-neo4j-p04-v1; DatasetVersion p04: + SHA256 canonical sorted used-source hashes + schema version; tidak invent published readiness. ID registry persis desain; edge ID hash tuple deterministik. Wrapper AvailableFrom berasal dari tanggal source/event atau 2026-10-01 untuk snapshot.

- [x] Tulis TestNeo4jSeedP04SourceGrounding, TestNeo4jSeedReferences, TestNeo4jSeedSafeAndDeterministic. Assert interaction IDs I0284/I0314/I0335, account P04 bukan deal ID, participant meeting K065/E06, event deal_ids=[], checksum/canonical IDs resolve, initialized arrays, quote Unicode exact, seed tanpa DELETE/DROP, uniqueness namespace/version. Assert current contact email bukan verified historical actor.
- [x] Run `go test ./repository -run TestNeo4jSeed -count=1`. Expected RED karena structs/artefak belum ada, bukan dependency/parser typo.
- [x] Implement private structs dan script: validate input hashes/header/native IDs/date/FK; fail explicit tanpa silently skipped rows. Select P04 records dan supporting entity/history rows saja. Quote source per field, raw file SHA256, tuple-based derived employment locator. Snapshot mutable values tetap bertanda snapshot. Native meeting participant evidence berasal dari peserta; historical email unresolved. Generator hanya menulis output di target eksplisit, bukan source; tidak menggunakan credentials/network.
- [x] Schema memakai IF NOT EXISTS dan composite uniqueness. Seed MERGE namespace/version/domain ID, ON CREATE SET untuk immutable payload. Existing manifest hash mismatch tidak overwrite atau mark ready. Ready query memverifikasi counts/payload hashes/references sebelum true; semua statement namespace-guarded. Output tidak menyertakan ready=true dalam preparation JSON.
- [x] Run script manual dengan DatasetPath ../Datasets relatif repo dan OutputDirectory backend/database/neo4j; rerun dan bandingkan actual output hashes. Pin artefak checksum-sensitive LF lewat .gitattributes. Expected bytes identik pada kedua run.
- [x] Run task tests/full suite. Expected GREEN. Catat preparation verified; live Cypher execution belum dijalankan. Jangan commit tanpa permintaan user.

## Task 2 — Historical/access-safe domain projection

**Files:** Create backend/repository/neo4j_projection.go, neo4j_projection_test.go.

**Consumes:** Task 1 storedContext/manifest/domain declarations, ValidateSnapshot, AccessScope membership. **Produces:** projectContext signature pada Shared storage shape, private fact/ref visibility helpers.

- [x] Tulis TestNeo4jProjectionDates dengan assertions:

```go
// raw dibaca dari artifact prepared source dalam test saja.
early, err := projectContext(raw, snapshotFor("2026-09-01"))
// err == nil; event entity IDs exactly I0284/I0314.
// JSON early tidak mengandung I0335 atau quote request reference.
// deal Stage/OwnerID/PotentialACVIDR/PlannedOutlets Value == nil.
late, err := projectContext(raw, snapshotFor("2026-09-22"))
// entity IDs exactly I0284/I0314/I0335; semua account-scoped.
// historical email chosen NodeID == nil; tidak ada consent invented.
current, err := projectContext(raw, snapshotFor("2026-10-01"))
// stage Negosiasi; owner E06; ACV 147000000; outlets 35 dengan bukti visible.
```

- [x] Tulis TestNeo4jProjectionScopeAndManifest (empty/analog-only tidak primary grant; wrong version/not ready/incomplete counts -> DataNotReady; tampered snapshot -> ContextMismatch), TestNeo4jProjectionReferenceClosure (future/nonauthorized evidence, future ValidTo, participant candidates/edge endpoints disembunyikan), TestNeo4jProjectionDoesNotMutate (early/current/repeated dan concurrent projection tidak mengubah raw). Expected RED missing projectContext.
- [x] Implement pure projection dengan initialized arrays, immutable input copy, per-field snapshot mask, evidence date/scope filter, event/node/edge/reference closure dan future-end masking. ContextID bukan auth. Meta.DataState/Limitations menyatakan coverage hanya P04. Illegal/corrupt stored refs -> DataNotReady, bukan silent empty hasil sehat.
- [x] Run `go test ./repository -run TestNeo4jProjection -count=1` lalu full suite/race. Expected GREEN, no raw mutation/future leaks dalam unit checks; belum proof live query.

## Task 3 — Adapter managed read, graph bounds dan cursor

**Files:** Create backend/repository/neo4j_repository.go, neo4j_driver.go, neo4j_queries.go, neo4j_cursor.go, neo4j_repository_test.go; modify backend/go.mod, create/update backend/go.sum.

**Consumes:** Task 2 visibleContext/projectContext dan interface existing. **Produces:** Neo4jConfig { URI, Username, Password, Database string }; NewNeo4jContextRepository(ctx context.Context, config Neo4jConfig) (*Neo4jContextRepository, error); (*Neo4jContextRepository).Close(ctx context.Context) error; implement ListFacts/FindFacts/ReadGraph/ReadTimeline/ReadEvidence persis context_repository.go. Tidak implement ActionRepository.

Private test seam contextReader func(ctx context.Context, snapshot models.SnapshotContext) (storedContext, error); constructor runtime selalu memasangnya ke official driver, tidak export fixture adapter. Query strings konstan/parameterized; driver types tidak keluar repository package.

- [x] Tulis compile assertions empat interfaces serta TestNeo4jRepositoryGraphBoundsAndFocus (default root account-context membawa relevant visible events; depth2 memperluas native participant neighborhood, depth invalid/bounds rejected, focus future/foreign denied, caps/closure/truncated deterministic), TestNeo4jRepositoryTimelineCursor (stable event_at/event_id, filters EventIDs/EventTypes/ActorNodeIDs/Statuses, limit1 paging, filter/context/scope tamper -> InvalidQuery/ContextMismatch), TestNeo4jRepositoryDealAndEvidenceErrors (before creation NotVisible; unauthorized AccessDenied; unknown within covered entity kind NotFound; known other deal outside slice DataNotReady).
- [x] Tulis TestNeo4jRepositoryCancellationAndDriverFailure (errors.Is Canceled/DeadlineExceeded, cause redacted, nil reader/not-ready bukan empty response), TestNeo4jQueriesReadOnlyAndParameterized (no runtime CREATE/MERGE/SET/DELETE; version/date/scope query params), TestNeo4jConfigValidation (neo4j+s URI/nonblank explicit DB/auth; password absent -> DataNotReady, no network attempt on invalid config). Expected RED missing adapter.
- [x] Pin dependency github.com/neo4j/neo4j-go-driver/v5@v5.28.4; keep go 1.22. Download via approved network path bila sandbox memblokir; tidak memasang server/Docker. Implement driver using NewDriverWithContext, per-call sessions AccessModeRead, ExecuteRead, ctx pada setiap Run/Collect, Close safely; no runtime import. Constructor config-validates dan VerifyConnectivity; manifest readiness diperiksa setiap managed operation.
- [x] Query manifest + covered records dalam managed transaction, per namespace/scope/as_of. Registry existence probes trusted/internal tetap tidak diekspos HTTP. Snapshot records boleh diambil untuk projection scalar mask, future dated events/evidence tidak perlu diambil. Validate actual counts/integrity readiness registry tanpa menganggap filtered counts sama dengan full seed counts. Decode payload errors -> DataNotReady; driver errors -> QueryFailed with Cause.
- [x] Implement methods: primary membership+native deal-account link+created cutoff; list options filter/paging; graph deterministic BFS account-context roots dengan strict depth/caps/focus/endpoint closure; timeline filters/stable sort; opaque cursor base64 canonical payload context/filter/order + validation server, bukan credential; evidence lookup scopes/dated refs. Reject unsupported filters, tidak ignore diam-diam. No analog/company widening pada slice primary.
- [x] Run `go test ./... -count=1`, `go vet ./...`, `go test -race ./... -count=1`, gofmt file baru saja, `git diff --check`. Expected GREEN; legacy/O2 source tidak berubah.

## Task 4 — Live gate dan handoff yang runnable

**Files:** Create backend/repository/neo4j_integration_test.go; backend/docs/neo4j-p04-runbook.md. Update plan progress/results hanya dengan evidence aktual.

- [x] Add opt-in integration test TestNeo4jP04Live (skip jika RELIO_NEO4J_INTEGRATION !=1; saat enabled missing config hard fail, bukan skip). Environment NEO4J_URI/USERNAME/PASSWORD/DATABASE plus RELIO_DATASET_VERSION; no secret output. Tests tidak menulis database, pakai seed yang dimuat eksplisit.
- [x] Assert live FindFacts, graph/timeline early vs late/current, evidence visible/future, consistent context IDs, refs, empty scope, seed registry/count readiness. Cara rerun seed manual dan count equality dicatat; tidak mengklaim live test memuat data otomatis.
- [x] Write runbook: preparation command, urutan schema/seed/verify di explicit Aura database, env via secure local channel, constructor O2 wiring example tanpa mengubah main, test commands dan exact coverage limitation P04. Read-only credential bila tersedia; schema load memakai credential preparation berbeda, tidak di browser frontend.
- [x] Run full tests/vet/race; report live SKIP atau actual PASS/FAIL secara terpisah. Independent fresh-context branch review satu kali; fix important tests RED/GREEN. Tidak publish tanpa user meminta, tidak menyebut graph sudah live sebelum seed/live acceptance.

## Review handoff

Plan self-review: pilihan user nomor1 konsisten dengan manual seed + read-only runtime. Task1 storage shape dikonsumsi Task2/3, Task2 projection dipakai Task3, Task4 membuktikan actual DB. Semua Review Focus punya assertions eksplisit. Source/identity ambiguity dijaga konservatif; empat interfaces saja, action/usage/full five deals dan HTTP tetap deferred. Tidak ada credential/instance assumed.

Eksekusi direkomendasikan Native: empat task berbagi storage/projection types; satu implementer + satu review akhir, tanpa check-in per task. Plan disetujui user dengan 'gas biar hemat token juga'; Native dieksekusi. Instruksi publish sebelumnya terbatas PR #4; tahap ini belum diizinkan commit/push/merge.

## Execution result

Empat task lokal selesai pada feat/neo4j-context-o1, belum commit/push/merge. Source-grounded seed dan empat read interfaces tersedia. Review independen dan satu pass tiga fixes terverifikasi; 63 top-level +72 subtests PASS, vet/race PASS, satu live SKIP. Cypher belum dieksekusi di Aura dan full live acceptance tetap pending; checklist di atas berarti local implementation/test gates sesuai expected skip, bukan bukti database ready. Rincian/limitations dan keputusan ada di neo4j-p04-runbook.md serta ignored plan ledger.
