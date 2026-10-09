# Neo4j Aura — graph nyata slice P04

Status: seed source-grounded dan adapter lokal; bukan acceptance Aura sampai live test PASS. Shared contract masih candidate untuk O2. Dataset FIX, database statis, preparation manual sekali; tidak ada pipeline/CSV startup/request.

## Yang tersedia

- Dataset coverage: DL-004/P04 saja. 1 deal, 7 nodes, 8 domain edges, 3 events, 27 field evidence units. Employment enrichment, actions, usage/support, akun analog dan empat demo deal lain belum dimuat.
- `backend/database/neo4j/schema.cypher`, `seed-p04.cypher`, `manifest-p04.json`, `context-p04.json`. JSON hanya preparation/tests, tidak menjadi database fallback runtime.
- `repository.NewNeo4jContextRepository`, `Neo4jConfig`, `Close`; implement DealFactsRepository, GraphRepository, TimelineRepository, EvidenceRepository. Bukan ActionRepository atau legacy DealRepository.
- Historical email actors/targets tetap raw/unresolved. Meeting K065/E06 native IDs; role history/nama current tidak diinvent. I0335 bukan consent/reference completed.

Pengukuran lokal artefak final: 27 domain evidence dalam compact JSON = 15.756 bytes UTF-8, context preparation = 39.162 bytes, seed = 124.655 bytes. Ini bukan cap model/evidence yang telah disepakati atau ukuran selected-action bundle; ActionRepository belum diimplementasikan.

## 1. Preparation lokal (tidak menghubungi Aura)

Dari root Relio, Go 1.22+ dan PowerShell:

```powershell
powershell -NoProfile -ExecutionPolicy Bypass -File ./tools/prepare-neo4j-p04.ps1 -DatasetPath ../Datasets -OutputDirectory backend/database/neo4j
```

Bypass hanya proses script ini, bukan perubahan execution policy global. Script memeriksa six used-source hashes against audited dataset, IDs/date/FK slice. Seluruh 15 source hashes telah diperiksa ulang sebelum desain. Source tidak diubah. Output directory eksplisit tidak boleh berada di source dataset. Rerun menghasilkan bytes identik. `.gitattributes` mengunci LF.

Manifest `ready:false` adalah artefak persiapan, bukan published DB readiness. Dataset version saat disiapkan:

`p04:8a7eecd15f97a7ca7fd770bbb917a6efb03ea004a9263e2af2634aff660b9c7a`

## 2. Load sekali ke target Aura yang eksplisit

Gunakan instance/database khusus Relio yang dipilih tim. Login Aura Query memakai credential preparation aman, bukan credential/browser frontend. Belum ada instance/secret yang diasumsikan atau dibuat oleh adapter.

1. Jalankan statement `schema.cypher` satu per satu di target database yang dipilih. Semua constraints `IF NOT EXISTS`.
2. Jalankan seluruh statement `seed-p04.cypher` berurutan, satu per satu atau fasilitas multi-statement UI bila tersedia. Jangan berasumsi satu query Bolt menerima seluruh file sekaligus.
3. Statement terakhir harus mengembalikan `ready:true`, `actual:46`, `links:8`. Bila false/no row/error, berhenti; jangan enable runtime atau mengubah manifest secara manual menjadi ready.
4. Seed rerun version/hash sama: counts tetap 46 records/8 adjacency links. MERGE/ON CREATE tidak overwrite existing payload. Hash mismatch tidak mengganti namespace atau mark ready; pakai fresh reviewed version/DB dan jangan DELETE data untuk menyembunyikan error.

Tidak menggunakan APOC, LOAD CSV URL, DROP, DELETE atau commercial write. Constraints/seed writes **tidak** dipanggil saat startup/request server. `RelioRecord` stores typed per-record JSON; `CONTEXT_LINK` menyimpan adjacency nyata dengan domain edge_type. Runtime memverifikasi payload SHA256/counts/adjacency sebelum projection. Hash bukan signature/auth; server/DB credential controls tetap wajib.

Visual inspeksi di Aura Query (internal only, bukan API historical):

```cypher
MATCH p=(a:RelioRecord)-[r:CONTEXT_LINK]->(b:RelioRecord)
WHERE r.dataset_version = 'p04:8a7eecd15f97a7ca7fd770bbb917a6efb03ea004a9263e2af2634aff660b9c7a'
RETURN p;
```

## 3. Konfigurasi server/live tests

Siapkan environment lewat kanal/file lokal aman, tidak Git/chat/frontend:

- NEO4J_URI: URI `neo4j+s://...` instance yang dipilih.
- NEO4J_USERNAME, NEO4J_PASSWORD: credential runtime read-only bila instance mendukung.
- NEO4J_DATABASE: nama database eksplisit, tidak menebak default.
- RELIO_DATASET_VERSION: version manifest yang benar-benar loaded/verified.

Jika menggunakan file lokal, simpan di `backend/.env` (sudah di-ignore Git). Adapter Go tidak otomatis membaca file tersebut: export ke environment proses sebelum tests/server, atau O2 menyediakan loader configuration. Jangan simpan sebagai file dengan nama lain tanpa mengecek aturan ignore.

Jangan copy credential ke runbook/public DTO/log/health. Adapter tidak expose driver types, raw payloads, queries atau Cause pada public errors.

## 4. Handoff constructor ke O2

```go
repo, err := repository.NewNeo4jContextRepository(ctx, repository.Neo4jConfig{
    URI: os.Getenv("NEO4J_URI"), Username: os.Getenv("NEO4J_USERNAME"),
    Password: os.Getenv("NEO4J_PASSWORD"), Database: os.Getenv("NEO4J_DATABASE"),
})
// O2 handles unavailable errors and owns Close lifecycle/HTTP wiring.
// Do not log the config/password or expose error Cause.
```

O2 membentuk AccessScope server (P04+DL-004 primary), NewSnapshotContext dan optional filter/options; bukan browser-provided authorization. Query `FindFacts`/`ReadGraph`/`ReadTimeline`/`ReadEvidence` memakai snapshot yang sama. Hash context adalah correlation, bukan credential.

Default graph depth1 memiliki account+deal anchors dan visible event neighbors; depth2 menambah native participant neighborhood. Focus tidak memberi akses future/foreign event. Timeline refs boleh lebih luas daripada graph terbatas; memakai focus pada snapshot yang sama untuk resolve. Page cursor terikat context/filter/order, bukan token izin.

Current commercial facts hanya known pada 2026-10-01. Sebelumnya snapshot_only/null; date-only source tidak diberi timestamp rekaan. Sebelum creation DL-004 -> not_visible; izin kosong -> access_denied; DB/manifest/slice belum siap -> data_not_ready, bukan empty success. Coverage-only list tidak boleh dianggap denominator seluruh prospek.

Private bounded namespace read mengambil raw P04 records dalam satu ExecuteRead transaction sebelum semua fields/references diproyeksikan. Tidak cocok diperluas langsung ke seluruh 229k source records tanpa selective/indexed query. Tidak mengembalikan record future/raw ke service.

## 5. Tests

Dari backend:

```powershell
go test ./... -count=1
go vet ./...
go test -race ./... -count=1
# Setelah schema/seed siap dan environment aman tersedia:
$env:RELIO_NEO4J_INTEGRATION = '1'
go test ./repository -run TestNeo4jP04Live -v -count=1
```

Live test default SKIP; enabled dengan config kurang harus FAIL, tidak skip atau mock success. Live test read-only, tidak mengisi database. Tests biasa menggunakan prepared source artifacts dan private reader double hanya untuk melewati transport eksternal; projection/filter/error behavior tetap production code.

Acceptance: cutoff 2026-09-01 I0284/I0314; 2026-09-22 menambah I0335, graph/evidence tanpa future refs, stage/owner/ACV/outlets historical null, current values dari source, consistent contexts, scope/access/bounds/cursor dan failures bukan empty success.

## Verification sesi

Final lokal: 63 top-level tests + 72 subtests PASS, 0 failure; go vet dan full race suite PASS. TestNeo4jP04Live default SKIP (1 test), bukan live PASS. Negative gate verified: enabled live test dengan config kurang FAIL, bukan skip. Preparation final rerun bytes identik; 27 proof excerpts/checksums diperiksa independen terhadap native source fields.

Satu review fresh-context menemukan tiga important issues; semuanya diperbaiki dalam satu test-first pass: seed ready membandingkan actual payload/adjacency, runtime decoder memeriksa sealed manifest record registry + native DL-004/P04 anchor, dan bounded focused graph mempertahankan event fokus. Test fokus/readiness gate diamati FAIL sebelum fix; registry test RED untuk decoder/registry yang belum ada lalu GREEN menolak payload+hash replacement. Full suite/vet/race dijalankan ulang. Cypher readiness-gate test adalah static statement check, bukan execution acceptance; seluruh file Cypher masih harus dijalankan dan live tests diulang pada Aura.

Instance Aura telah dikonfirmasi ada oleh user; konfigurasi lokal akan disiapkan. Aura belum dihubungi, schema/seed belum dieksekusi dan live test belum PASS. HTTP wiring/deployment/JEV/Copilot bukan deliverable tahap ini. Belum commit/push/merge.
