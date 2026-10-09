# Relio — Handoff kontrak Go O1 ke O2

Tanggal: 10 Oktober 2026. Branch: `feat/context-contract-o1`. Base: `57420ac`.

**DRAFT — deklarasi Go candidate, menunggu review Orang 2.** Prinsip F02–F08 telah di-ACK O1; ini bukan persetujuan O2 atas source baru, database siap, atau MVP complete.

## Hasil tahap ini

- Tipe domain, lima interface read-only, typed errors dan helper pure telah diimplementasikan.
- Kontrak legacy `models.Deal`/`DealRepository`, main, routes, controllers, services, go.mod dan dokumen arsitektur/draft bersama tidak diubah.
- Dataset bisnis tidak diubah atau dimuat saat request/startup. Tidak ada adapter fixture produksi, ETL, driver, migration, atau pilihan instance DB baru.
- Tests terisolasi memakai **DATA UJI / TEST-* IDs**, bukan query dataset kompetisi.

Acuan: [ACK O1](data-repository-contract-ack-o1.md), [review O2](data-repository-contract-response-o2.md), [plan tahap pertama](superpowers/plans/2026-10-10-context-contract-foundation.md).

## File yang dapat direview/dikonsumsi O2

| Area | File | Kegunaan |
|---|---|---|
| Nullable/provenance | [fact.go](../models/fact.go) | `Fact[T]`, validation state/basis/value/evidence nonempty. |
| Context/scope | [context.go](../models/context.go), [access_scope.go](../models/access_scope.go) | Context/meta/bounds dan membership preconditions. |
| Tanggal | [date.go](../models/date.go) | Strict calendar parsing; next-day query boundary; embedded timezone data untuk Windows. |
| Deal | [deal_facts.go](../models/deal_facts.go) | Additive facts/page/detail; tidak memaksa unknown ke model lama. |
| Graph/timeline | [graph.go](../models/graph.go), [event.go](../models/event.go) | Typed nodes/edges/participants/event refs; bukan implementasi traversal. |
| Evidence/actions | [evidence.go](../models/evidence.go), [action.go](../models/action.go) | Source locators, occurrences, policy facts, candidate/selected bundle types. |
| Read interfaces | [context_repository.go](../repository/context_repository.go) | DealFactsRepository, GraphRepository, TimelineRepository, EvidenceRepository, ActionRepository. |
| Repository errors | [context_error.go](../repository/context_error.go) | 11 kategori, errors.As/Is + Unwrap; Cause/entity IDs/issues tidak diserialisasi. |
| Snapshot guards | [snapshot.go](../repository/snapshot.go) | NewSnapshotContext/ValidateSnapshot; canonical scope hash. |
| Query guards | [query_bounds.go](../repository/query_bounds.go) | Graph defaults/max dan page limits; tidak menjalankan traversal. |
| Contract fixtures | [context_contract_cases.json](../repository/testdata/context_contract_cases.json), [contract_sources.json](../repository/testdata/contract_sources.json) | DATA UJI; actual fixture-file checksum/locator checks, bukan manifest bisnis. |

### Method signatures

Semua method menerima `context.Context` dan `models.SnapshotContext`:

- `ListFacts(ctx, snapshot, options) (models.DealFactsPage, error)`.
- `FindFacts(ctx, dealID, snapshot) (models.DealFactsResult, error)`.
- `ReadGraph(ctx, dealID, snapshot, options) (models.GraphResult, error)`.
- `ReadTimeline(ctx, dealID, snapshot, options) (models.TimelineResult, error)`.
- `ReadEvidence(ctx, evidenceID, snapshot) (models.EvidenceResult, error)`.
- `ReadActionCandidates(ctx, dealID, snapshot, options) (models.ActionCandidatesResult, error)`.
- `ReadSelectedActions(ctx, dealID, snapshot, selectedTemplateIDs, relevanceRuleVersion) (models.SelectedActionsResult, error)`.

Selected atomic retrieval dan kandidat revalidation adalah kewajiban adapter **yang belum diimplementasikan**. Method terakhir boleh menginspeksi satu ID; validasi 2–4 unik untuk compare tetap milik O2.

## Semantik yang perlu dipertahankan consumer

1. Versi source sekarang `context-contract-v0.1-candidate`, bukan final bersama. Dataset version harus berasal dari published preparation manifest; NewSnapshotContext tidak membuat atau memverifikasi manifest tersebut.
2. `Fact.Value` tidak memakai omitempty. Known membutuhkan pointer/value dan evidence nonempty; unknown/ambiguous/snapshot_only tidak memilih value. Known zero/false berbeda dari null. Validate tidak membuktikan evidence existence/access/history atau domain-specific scalar validity.
3. JSON tags domain adalah serialization candidate untuk fixtures/handoff, **bukan pembekuan DTO HTTP**. O2 masih membuat DTO. Slice/map output valid harus diinisialisasi menjadi []/{}; structs mentah zero-value **tidak otomatis dinormalisasi**.
4. `AllowedAnalogAccountIDs` adalah tambahan eksplisit dari ACK. Izin primary deal memerlukan deal dan account membership; analog-only tidak memberi primary permission. Membership helper bukan autentikasi atau bukti FK native; adapter tetap memeriksa actual relationship dan scope setiap result/reference.
5. `SnapshotContext.Access` tidak diserialisasi. Context hash bukan credential/signature; jangan menerima Access atau ContextID dari browser sebagai otorisasi. NewSnapshotContext dipanggil setelah O2 membentuk scope server.
6. Date parser menerima tanggal kalender umum, termasuk planned date sesudah cutoff; **snapshot constructor** menolak as_of sesudah 2026-10-01. Tidak trim/clamp. Empty/duplicate/default absent HTTP tetap validasi O2.
7. Untuk as_of 2026-10-01, next-day exclusive di Asia/Jakarta adalah 2026-10-01T17:00:00Z. Nilai ini hanya query bound, tidak menjadi fabricated source/event timestamp.
8. Konvensi employment selesai 2026-08-15 -> exclusive valid_to 2026-08-16 diuji; raw date tidak diubah. Mask future validity/outcome/quotes dan semua projection historical tetap pekerjaan adapter berikutnya.
9. Graph options zero berarti absent default (1/150/300); max 2/150/300, invalid ditolak bukan clamp. Page default/max 50. Focus existence/ownership/access dan cursor context/filter binding belum diimplementasikan.
10. Typed errors bukan HTTP response. O2 memilih status/code/message/redaksi aman; tidak mengekspor Cause/Issues mentah. Error categories berbeda dari sentinel legacy, tanpa implicit compatibility bridge.

## Verification aktual

Go resmi dipasang atas permintaan eksplisit user melalui winget. Installer hash diverifikasi winget; `go version` mengonfirmasi `go1.27.0 windows/amd64`. Tidak mengubah floor `go 1.22` dalam go.mod; versi 1.22 tepat **belum diuji**.

| Pemeriksaan | Hasil sesi ini |
|---|---|
| Baseline existing main/controllers/services/routes | PASS sebelum source produksi baru. |
| RED sebelum implementasi | Fact/domain, repository errors/interfaces, date, snapshot/scope, bounds: expected missing declarations. Fixture tests: expected testdata missing. |
| `go test ./... -count=1` | PASS setelah fix review: 44 top-level tests + 50 subtests, 0 failures; 25 top-level tests baru O1. |
| `go vet ./...` | PASS. |
| `go test -race ./... -count=1` | PASS pada Windows environment ini. |
| Formatting | gofmt diterapkan hanya pada source Go baru. File legacy tidak diformat ulang. |
| `git diff --check` | PASS untuk tracked diff; source baru tetap untracked sampai user commit. |
| Perubahan existing tracked files | Tidak ada; semua deliverables source O1 file tambahan. |
| Live DB / JEV / HTTP migration / E2E | Tidak dijalankan atau diklaim oleh tahap ini. |

Default GOCACHE tidak writable dari sandbox agent. Untuk tests agent digunakan **process-only** cache di ignored plan workspace; tidak mengubah konfigurasi Go global.

PowerShell dari backend, sesudah membuka terminal baru agar PATH installer terbaca:

```powershell
go version
go test ./... -count=1
go vet ./...
go test -race ./... -count=1
```

Jika sandbox agent masih perlu cache khusus:

```powershell
$env:GOCACHE = [IO.Path]::GetFullPath((Join-Path (Get-Location).Path '..\.superpowers\sdd\2026-10-10-context-contract-foundation\go-cache'))
& 'C:\Program Files\Go\bin\go.exe' test ./... -count=1
```

Jangan mengubah environment user permanen hanya untuk workaround cache tests ini. Race tests memakai C toolchain yang telah tersedia, bukan instalasi compiler baru oleh O1.

## Acceptance coverage dan batas

- Teruji: Fact null vs zero/false, ambiguity serialization, required fixture collections, category/error safety/wrapping, scope/context self-consistency, strict dates/bounds dan fixture references/source checksum.
- Fixture menegaskan request10%/approval10%/request14% terpisah; approved14%/consent/outcome tetap unknown. Satu candidate tidak dipad menjadi dua.
- Tests **tidak** menjalankan database graph/timeline/evidence/actions, memfilter seluruh future fields, melakukan identity resolution asli, membuktikan approval/consent asli, atau selected batch atomic lookup. T01–T15 adapter acceptance lengkap masih tertunda.
- Semua collections initialized dalam fixture, bukan janji auto-normalization seluruh domain responses.
- Metadata TEST dalam fixture bukan dataset_version atau context_id bisnis siap pakai.

## Hasil review kode independen

Reviewer fresh-context menyatakan siap untuk review tim, tanpa temuan critical. Celah coverage nested references yang semula dinilai minor diregrade menjadi important: fixture dapat lolos walau evidence pada Fact atau node/candidate pada ParticipantRef putus. Helper **test-only** kini menelusuri references tersebut; lima mutation cases menolak dangling IDs. Test ditulis lebih dahulu dan gagal karena helper belum tersedia (RED), lalu lulus setelah implementasi (GREEN); full tests, vet dan race dijalankan ulang dan lulus. Tidak ada perubahan perilaku runtime dari fix ini.

Reviewer tidak mengesahkan adapter yang belum ada, HTTP/auth, normalisasi otomatis semua responses, registry bisnis, atau policy application. Batas tersebut tetap eksplisit: O1 mengerjakan adapter/source-grounded acceptance pada tahap berikutnya; O2 mengerjakan public DTO/auth/HTTP/application policy. Approval unknown pada occurrence request tetap dapat berdampingan dengan occurrence approval yang terpisah, bukan inheritance approval ke request baru.

Ini review kode independen, **bukan persetujuan Orang 2**. Tidak ada deferred minor dari review tersebut setelah satu fix pass.

## Review minimum O2 dan pekerjaan paralel berikutnya

1. Review deklarasi/nama method, tambahan scope analog, candidate version dan domain serialization vs public DTO. Catat ACK/koreksi sebelum migrasi consumer.
2. O2 dapat membangun test-only stubs, DTO/service/controller facts, temporal/access HTTP validation, serta error translation. Jangan mengubah current endpoints secara diam-diam menjadi rich contract tanpa tests/documentation.
3. O1 berikutnya: pilih engine/instance bersama, siapkan schema/data statis sekali, manifest/registry, implement adapter read-only dan tests source-grounded.
4. Slice nyata pertama DL-004/P04: snapshot 2026-09-01 account interactions I0284/I0314; 2026-09-22 tambah I0335. Itu expected source observations dari audit lama, **bukan query yang sudah dijalankan tahap ini**. Source checksum perlu diverifikasi ulang sebelum live acceptance.
5. F01 engine/instance dan F07 cap evidence tetap terbuka. Tidak menggantikan keduanya dengan fixture, vendor asumsi atau angka cap rekaan.

Pada persiapan publikasi PR, line ending contract_sources.json dikunci LF lewat testdata/.gitattributes agar Git Windows autocrlf tidak mengubah raw-byte checksum saat clone/checkout. Tidak mengubah setting Git global atau dataset bisnis. Pemeriksaan attribute dijalankan RED sebelum aturan tersedia, lalu GREEN sesudahnya, dan tests dijalankan ulang.

Review kode independen selesai dan fix terverifikasi. Publikasi branch/PR diminta user; status tetap DRAFT sampai Orang 2 menyetujui deklarasi/kontrak consumer.
