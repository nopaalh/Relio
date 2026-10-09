# Context Contract Foundation Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox syntax for tracking.

**Goal:** Serahkan deklarasi Go additive dan test-only fixtures agar O2 dapat membangun consumer API tanpa menunggu database atau mendefinisikan ulang mapping O1.

**Architecture:** Domain types langsung di backend/models; read interfaces dan typed errors langsung di backend/repository. Pertahankan seluruh legacy consumer. Helper pure menguji null/provenance, tanggal dan snapshot/scope; tidak melakukan I/O bisnis atau menyajikan fixtures di runtime.

**Tech Stack:** Go 1.22+, standard library; tidak menambah driver/database/dependency pada tahap ini.

**Spec:** [draft O1, terutama A–B dan T01–T15](../../data-repository-contract-draft.md), [review O2 F01–F08](../../data-repository-contract-response-o2.md), [ACK O1](../../data-repository-contract-ack-o1.md). Review minimum desain telah diberikan O2; shape deklarasi Go berikut tetap candidate untuk review consumer.

## Global Constraints

- Dataset FIX; database bisnis STATIS, disiapkan sekali; tanpa pipeline ETL, connector, scheduler, refresh atau CSV parsing pada request/startup.
- Maksimum as_of `2026-10-01`, seluruh hari `Asia/Jakarta`; internal interval `[valid_from, valid_to)`.
- Missing bukan zero/false/rejected/resolved; tidak membuat recorded_at, stage history, consent, approval atau action baru.
- `DealRepository` dan `models.Deal` existing tidak diubah; main/routes/controllers/services milik O2.
- Tidak membuat internal/ atau cmd/. Tidak menetapkan DB engine/instance atau cap evidence rekaan.
- Nama/type shared masih candidate untuk review O2; DTO/HTTP, scoring, JEV, cache dan Copilot tidak diimplementasikan di sini.
- Fixtures DATA UJI hanya di tests/testdata. Tidak mengekspor adapter fixture yang dapat di-wire sebagai database bisnis.
- Tidak commit/push/merge atau memasang toolchain tanpa permintaan tersendiri.

## Review Focus

- Empty/null JSON vs known zero/false: assert value:null berbeda dari value:0/false, tanpa omitempty pada value.
- Tanggal kosong, invalid, leap day dan seluruh hari cutoff: parsing tidak clamp; next-day exclusive setara 2026-10-01T17:00:00Z untuk as_of 2026-10-01.
- Scope kosong/analog-only, urutan allowlist dan scope berubah: tidak memberi izin tersirat; context stabil untuk himpunan sama dan berubah untuk izin berbeda.
- Error wrapping/cancellation dan data sensitif: errors.Is/As bekerja; Error()/serialization tidak menampilkan Cause/secret/entity IDs.
- Batch/graph refs dan states yang inconsistent: fixtures punya endpoints/resolvable refs; unknown approval bukan default false, dan account-only tidak punya deal IDs.

## Preparation gate

- [x] Baca kontrak, review O2, READMEHack dan source existing; catat ACK tanpa menimpa draft/arsitektur.
- [x] Periksa working tree main 57420ac bersih sebelum dokumen ini.
- [x] User review rencana ini sebelum menulis source. Eksekusi rekomendasi: native/inline oleh O1; subagent tidak diperlukan untuk paket kecil yang saling berbagi types.
- [x] Pilih branch/workspace implementasi. Checkout saat ini main, bukan linked worktree. Tanyakan preferensi worktree sebelum membuat worktree; jika bekerja di checkout ini gunakan branch `feat/context-contract-o1`, setelah memastikan changes milik user tidak tertimpa.
- [x] Temukan executable Go 1.22+ yang sudah tersedia dan jalankan baseline `go test ./...`. PATH agent saat audit belum mengenali `go`; jangan menyatakan baseline lulus atau install otomatis. Jika toolchain belum tersedia, laporkan gerbang ini sebelum TDD.

## Task 1 — Nullable facts dan domain declarations

**Files:** Create `backend/models/fact.go`, `context.go`, `deal_facts.go`, `graph.go`, `event.go`, `evidence.go`, `action.go`; tests `backend/models/fact_test.go`, `domain_contract_test.go`.

**Interfaces:** Deklarasi field/type mengikuti persis block B draft, dipindahkan ke package models. `Date` underlying string; `Fact[T]` memakai Value *T, State/TemporalBasis string dan EvidenceIDs/Limitations []string. Add `AllowedAnalogAccountIDs []string` pada `AccessScope` sesuai ACK. Tidak menambahkan score/suitability atau HTTP handler.

Produces `Fact[T].Validate() error` untuk states known/unknown/ambiguous/snapshot_only dan bases event/interval/snapshot/undated. Known memerlukan non-nil value dan setidaknya satu nonempty evidence ID; unknown/ambiguous/snapshot_only tidak memilih value. Nil array dapat ada pada input Go mentah, tetapi output fixture valid wajib initialized [] dan bukan JSON null. Fact Value tidak memakai omitempty.

- [x] Tulis `TestFactNullZeroAndFalseRemainDistinct`, `TestFactKnownNeedsValueAndEvidence`, `TestFactRejectsUnknownWithValue`, `TestFactRejectsInvalidStateAndBasis`. Assert JSON unknown value:null; known int zero value:0; known bool false value:false; error pada inconsistent states. Test domain fixture memiliki semua arrays initialized.
- [x] Jalankan `go test ./models -run 'TestFact|TestDomain' -count=1`; pastikan red karena contract belum tersedia, bukan typo/dependency.
- [x] Implement declarations dan minimal Validate yang memenuhi assertions. Gunakan domain JSON tags snake_case untuk Fact sesuai draft; public DTO domain lainnya tetap pekerjaan O2, bukan deklarasi final HTTP.
- [x] Jalankan tests task dan `go test ./...`; catat hasil aktual.

## Task 2 — Typed repository errors dan read interfaces

**Files:** Create `backend/repository/context_repository.go`, `context_error.go`, `context_error_test.go`, `context_repository_test.go`. Tidak modify deal_repository.go.

**Interfaces produced:**

```go
type DealFactsRepository interface {
    ListFacts(context.Context, models.SnapshotContext, models.DealListOptions) (models.DealFactsPage, error)
    FindFacts(context.Context, string, models.SnapshotContext) (models.DealFactsResult, error)
}
type GraphRepository interface {
    ReadGraph(context.Context, string, models.SnapshotContext, models.GraphOptions) (models.GraphResult, error)
}
type TimelineRepository interface {
    ReadTimeline(context.Context, string, models.SnapshotContext, models.TimelineOptions) (models.TimelineResult, error)
}
type EvidenceRepository interface {
    ReadEvidence(context.Context, string, models.SnapshotContext) (models.EvidenceResult, error)
}
type ActionRepository interface {
    ReadActionCandidates(context.Context, string, models.SnapshotContext, models.CandidateOptions) (models.ActionCandidatesResult, error)
    ReadSelectedActions(context.Context, string, models.SnapshotContext, []string, string) (models.SelectedActionsResult, error)
}
```

Standalone string adalah dealID, kecuali ReadEvidence evidenceID. Selected menerima selectedTemplateIDs lalu relevanceRuleVersion. Parameter bernama pada source nyata.

ErrorCode constants, LookupIssue dan RepositoryError mengikuti B draft: NotFound, NotVisible, DataNotReady, AmbiguousReference, QueryFailed, InvalidSnapshot, InvalidQuery, AccessDenied, ContextMismatch, InvalidSelection, BundleLimitExceeded. `Error() string` hanya code dan `Unwrap() error` mengembalikan Cause. Cause serta ID/issues tidak diserialisasi otomatis menjadi HTTP; gunakan json:"-" untuk internal details. O2 menerjemahkan/redaksi.

- [x] Tulis `TestRepositoryErrorPreservesCause` (errors.Is DeadlineExceeded/Canceled), `TestRepositoryErrorSafeMessage`, `TestRepositoryErrorAsThroughWrap`, `TestRepositoryErrorDistinctCodes`; masukkan secret-like fake Cause dan pastikan tidak muncul di Error/JSON.
- [x] Jalankan `go test ./repository -run TestRepositoryError -count=1`; verifikasi red.
- [x] Implement errors/interfaces; compile-time assertions terhadap stub **hanya di _test.go**, tanpa driver/query text pada interface. Test signature legacy tetap tersedia tanpa perubahan.
- [x] Jalankan task tests lalu full suite.

## Task 3 — Pure date, snapshot dan bounds guards

**Files:** Create `backend/models/date.go`, `date_test.go`; `backend/repository/snapshot.go`, `snapshot_test.go`, `query_bounds.go`, `query_bounds_test.go`.

**Interfaces:**

- `models.ParseDate(raw string) (models.Date, error)` strict YYYY-MM-DD, tidak trim/clamp atau menerima timestamp.
- `models.Date.NextDay() (models.Date, error)` kalender +1, bukan source timestamp baru.
- `models.Date.NextDayStart(zone *time.Location) (time.Time, error)` helper query-only; tidak disimpan sebagai event/source timestamp.
- `repository.NewSnapshotContext(asOf models.Date, datasetVersion string, access models.AccessScope) (models.SnapshotContext, error)` menetapkan candidate ContractVersion dan CalendarZone; menolak empty/invalid/>cutoff date, empty datasetVersion/empty allowlist members. Tidak memberi default absent HTTP (O2).
- `repository.ValidateSnapshot(snapshot models.SnapshotContext) error` memeriksa date/cutoff, versi/zone, context hash dan scope. Context hash canonical JSON + SHA256 atas tanggal/versi/zone dan sorted/deduplicated allowlists termasuk analog/company. ContextID opaque, bukan credential. Tidak mutate slices input.
- `repository.NormalizeGraphOptions(options models.GraphOptions) (models.GraphOptions, error)` zero bounds menjadi defaults 1/150/300; negatif atau >2/150/300 ditolak; empty non-nil FocusEventID ditolak. Validasi focus existence/scope milik adapter.
- `repository.NormalizePageLimit(limit int) (int, error)` zero ->50; 1..50 diterima; selainnya invalid_query.

- [x] Tulis date tests format invalid/empty, Feb29 valid/invalid, rollover, next-day exclusive Jakarta cutoff; employment raw end 2026-08-15 -> valid_to 2026-08-16 tanpa memodifikasi raw.
- [x] Run date tests red; implement minimal date helpers; rerun green.
- [x] Tulis `TestSnapshotRejectsInvalidAndFuture`, `TestContextIDStableAcrossScopeOrder`, `TestContextIDChangesWithSnapshotAndPermission`, `TestValidateSnapshotRejectsTampering`, `TestEmptyScopeAndAnalogPermissionRemainSeparate`; scope kosong valid tapi grants none, analog-only bukan primary account grant. Tidak menentukan actual allowed demo IDs di production.
- [x] Run snapshot tests red; implement helpers; rerun green.
- [x] Tulis bounds default/negative/above max tests, run red, implement, run green/full suite.

## Task 4 — Test-only handoff fixtures dan review package

**Files:** Create `backend/repository/testdata/context_contract_cases.json`, `contract_fixtures_test.go`; `backend/docs/context-contract-go-handoff.md`.

**Interfaces:** Fixtures DATA UJI mereferensikan declarations Task1/2/3; tidak ada in-memory provider exported di production. Fixtures terpisah dari source-cases-draft dataset. Cases memakai TEST-* IDs, date-only events, account-only links, ambiguous participant node:null, known-zero vs unknown, request10%/approval10%/request14% dengan approved14% unknown. Policy flags/execution tetap O2.

- [x] Tulis fixture validation tests: unique IDs, semua edge endpoints resolve, event/evidence refs resolve, stable `(event_at,event_id)` expectation, empty arrays [] dan facts valid, account-only deal_ids [] dan approval tidak melekat ke request lain. Assertions kegagalan mengubah request14% menjadi approved atau node ambiguous menjadi verified harus jelas. Tidak mengklaim database historical filtering sudah teruji oleh fixture statis.
- [x] Run fixture test red; tambah fixture JSON berlabel DATA UJI; rerun green. HTTP stubs khusus consumer tetap milik O2.
- [x] Jalankan `gofmt` pada file Go baru, `go test ./... -count=1`, `go vet ./...`, `go test -race ./...` bila toolchain Windows mendukung; catat skip/failure, tidak menyamakan dengan pass.
- [x] Jalankan `git diff --check` dan periksa daftar perubahan: hanya file O1 baru. Tidak mengubah source legacy, dokumen shared, API, go.mod atau dataset.
- [x] Tulis handoff: deklarasi/versi candidate, tambahan scope analog, cara tests, hasil aktual dan batas yang belum dikerjakan. Serahkan ke O2 untuk review sebelum consumer migrasi.

## Deferred gates, bukan pekerjaan yang diam-diam dianggap selesai

Adapter, static schema/data preparation, canonical registry, source checksum revalidation, graph/timeline/evidence temporal projection, actual candidate extraction/relevance, selected atomic retrieval, cursor/query binding dan live DB acceptance T01–T15 memerlukan tahap berikutnya. Query helper/unit tests bukan bukti tidak ada future leakage di database nyata.

F01 engine/instance dan F07 evidence cap tetap terbuka. Tidak ada JEV/Copilot/HTTP integration atau deployment claim dari tahap ini.

## Execution record — 10 Oktober 2026

User approved this plan with "gas", native execution on feat/context-contract-o1 without extra worktree. User subsequently requested installation of Go; official Go 1.27.0 was installed and verified. Process-only GOCACHE under ignored plan workspace avoids sandbox cache write denial; no global Go config change.

All four tasks implemented with observed RED/GREEN, full tests/vet/race passing. Source and fixtures remain uncommitted file additions; no legacy/O2 source changes. Source checksum tests use separate contract_sources.json DATA UJI, not competition records. Scope membership helpers are preconditions only; domain tags are candidate serialization, not final HTTP DTO.

See [handoff](../../context-contract-go-handoff.md) for actual verification, limitations and O2 review requirements. Independent fresh-context code review completed. Nested Fact evidence and ParticipantRef node/candidate/evidence coverage was strengthened in one test-first fix pass; final full run passed 44 top-level tests plus 50 subtests, with vet/race also passing. This review cannot substitute for O2 contract approval.
