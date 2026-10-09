# Neo4j Aura — slice graph nyata O1

Status: desain disetujui user melalui "gas nomor 1", plan/Native disetujui melalui "gas biar hemat token juga" pada 10 Oktober 2026. Implementasi lokal berjalan; belum koneksi/load/live acceptance Aura. Deklarasi shared Go tetap candidate untuk review O2; tidak membekukan DTO atau mengubah interface.

## Tujuan dan batas

Prioritaskan satu slice nyata DL-004/P04: query graph, timeline dan evidence dari database yang dipersiapkan sekali. Bukan fixture production atau respons hardcoded. Dataset FIX; database bisnis STATIS; tidak ada pipeline ETL, connector, scheduler, refresh, atau pembacaan CSV pada startup/request API. Tahap ini bukan seluruh MVP.

Gunakan models dan repository interfaces yang sudah ada, langsung di backend/. O2 tetap memiliki HTTP, service orchestration, scoring, JEV, Copilot, auth dan wiring. Tidak membuat internal/ atau cmd/. Actions, usage, support dan commercial policy extraction ditunda; tidak mengembalikan kandidat kosong seolah sudah diimplementasikan.

## Pilihan dan keputusan desain

1. Pilihan utama: artefak seed Cypher statis dari source yang diverifikasi, dijalankan manual sekali; adapter Go hanya ExecuteRead. Cepat diperiksa di Aura Query, tidak perlu server import tambahan. Generasi artefak offline bukan pipeline, tidak dipanggil aplikasi.
2. Importer Go yang langsung menulis Aura: lebih mudah batching penuh tetapi memperlebar dependency dan risiko permission/data overwrite. Tidak diperlukan untuk slice awal.
3. Aura LOAD CSV dari URL publik: memerlukan hosting/access dataset tambahan. Tidak dipilih; source tetap lokal dan hanya artefak terpilih dipersiapkan.

Belum ada URI/database/credential yang dikonfirmasi. Persiapan source/seed dan unit tests bisa berjalan tanpa secret. Memuat seed ke instance dan live integration memerlukan target database yang dinyatakan user; tidak memilih instance sembarang atau menghapus data existing.

## Source-grounded acceptance

Checksum SHA256 seluruh 15 file Datasets sudah diperiksa ulang sesi ini; seluruhnya sama dengan expected_input_hashes pada source-cases-draft.json. Ini verifikasi input, bukan manifest database siap.

- crm_deals.csv DL-004: account P04, dibuat 2026-08-05; stage/owner/outlets/ACV/status adalah snapshot 2026-10-01. Historical stage tidak disimpulkan dari stage_sejak atau body meeting.
- interactions.jsonl I0284 2026-08-10, I0314 2026-08-28, I0335 2026-09-22: native account P04, tidak ada deal_id.
- Meeting I0284/I0314 peserta K065 dan E06 adalah native IDs. Nama/jabatan/current email bukan otomatis facts historis.
- I0335 email exact current contact match tidak dianggap verified historical identity. Raw email disimpan; tidak mengidentifikasi kalimat Direktur Utama sebagai K028 otomatis. Tidak membuat alias historis.
- Employment raw selesai diubah menjadi exclusive next-day internal sesuai konvensi F03; raw source tetap utuh. Future end/next role disembunyikan dari historical output.

Gunakan source metadata crm_accounts, crm_contacts, employees dan employment hanya untuk entity/provenance yang benar-benar diperlukan. Tidak membuat account ID untuk organisasi luar.

## Schema dan IDs

Dedicated dataset version namespace; tidak memakai Neo4j internal elementId sebagai ID domain.

- Node IDs: deal:DL-004, account:P04, contact:K065, employee:E06, event:interaction:I0284 (dan dua event lain).
- Event IDs: event:interaction:<native interaction_id>; entity_id tetap native interaction_id.
- Evidence IDs: ev:<source basename>:<native record ID>:<field>. Native source_record_id tidak diganti ID evidence. History tanpa row ID memakai record_id_kind=derived dengan tuple sumber + hash deterministik, bukan mengaku native.
- Edge IDs diturunkan deterministik dari relation type, source, target, source locator/claim dan schema version; tidak memakai random UUID.
- Label Neo4j RelioNode/RelioEvent/RelioEvidence/RelioDeal/RelioDataset; scoped uniqueness pada (dataset_version, domain ID). Domain type menjadi property/label tambahan yang dibatasi registry.
- Relasi graph inti: deal ACCOUNT_CONTEXT account, event CONCERNS_ACCOUNT account, event HAS_PARTICIPANT person untuk native meeting IDs. Tidak membuat event->deal dari shared account. Relasi employment terpisah dengan interval dan provenance jika digunakan.
- HAS_EVIDENCE dan event/edge evidence registries menjaga reference resolution. Semua evidence penting field-scoped, checksum file, raw source date, quote asli, scope account/deal eksplisit. Tidak mengembalikan seluruh reply thread.

Source snapshot facts memiliki available_from 2026-10-01, dated event facts memakai tanggal sumber. Jangan invent recorded_at, observed_at, timestamp atau riwayat stage. Planned dates tidak diperlakukan sebagai completed events.

## Preparation dan readiness

Schema constraint menggunakan IF NOT EXISTS dan seed memakai MERGE pada namespace/version/ID. Tidak menggunakan DELETE/DETACH DELETE, DROP, overwrite source, atau memilih database dari konfigurasi tebakan. Rerun version sama tidak menggandakan graph; perbedaan source/seed version harus ditolak, bukan overwrite existing version diam-diam.

Manifest memuat schema version, actual source hashes, deterministic dataset_version, scope coverage (hanya slice P04), counts dan ID registry. Ready ditandai hanya setelah constraints/data/reference/count checks berhasil. Semua write hanya lewat seed eksplisit; runtime adapter tidak menginisialisasi database.

Runtime menerima config URI/user/password/database dari environment server. Aura memakai neo4j+s TLS; password tidak dimasukkan Git/browser/log/health. Driver private pada adapter; service tidak melihat Cypher atau tipe driver. Gunakan official driver compatible dengan floor Go yang diperiksa sebelum dependency dipilih; jangan bump go.mod floor otomatis.

## Read adapter

Implement empat interface awal: DealFactsRepository, GraphRepository, TimelineRepository, EvidenceRepository. ActionRepository tidak diklaim dipenuhi. Tidak mengubah legacy DealRepository atau consumer O2.

- Setiap method: context cancellation, ValidateSnapshot, published manifest/version readiness, server scope, lalu managed ExecuteRead transaction. Dataset unknown/belum ready bukan empty success.
- Deal account/created anchor diperiksa melalui native FK. Snapshot-only commercial fields menjadi snapshot_only/null sebelum 2026-10-01. Current names/roles juga tidak diproyeksikan historis tanpa dated evidence.
- Graph: anchor deal + account-context, event cutoff inclusive, valid_from <= as_of < valid_to; future valid_to tidak dibocorkan. Depth 1/2, max 150/300; deterministic ordering, endpoint closure, truncated/reasons eksplisit. Focus wajib related/visible/authorized, bukan bypass scope/waktu/bounds.
- Timeline: sort (event_at,event_id), filters yang dikenal, limit <=50, cursor terikat context dan filter. Account-only event tetap deal_ids=[]; current account context bukan native deal evidence.
- Evidence: lookup per field/record, tanggal dan scope diperiksa; future tidak dikembalikan, analog/company permission tidak menjadi primary access. Event/edge references difilter ke konteks visible/authorized.
- Graph, timeline dan evidence memakai context_id/meta yang sama. Empty collections selalu []; unknown bukan zero/false. Tidak menawarkan source IDs/refs yang tidak bisa di-resolve pada scope/snapshot yang sama.
- Dataset version dengan coverage hanya P04 tidak melayani deal lain sebagai source-not-found; data_not_ready dengan limitation pada coverage tersebut. Perluasan lima demo deal menjadi slice berikutnya setelah acceptance awal.
- Error categories mempertahankan not_found, not_visible_at_snapshot, access_denied, data_not_ready, ambiguous_reference dan query_failed; Cause tetap Unwrap tanpa diekspos. HTTP mapping tetap O2.

## Tests dan handoff

Tulis tests RED sebelum implementasi untuk IDs/provenance/graph projections, readiness/scope/cursor/bounds dan mapper. Test doubles hanya di tests. Expected P04 native IDs diassert dari seed hasil sumber, bukan mock-only dataset business readiness.

Acceptance nyata setelah seed dimuat:

1. 2026-09-01 timeline berisi I0284/I0314 saja; graph/evidence tidak mengandung I0335, quote maupun derived future refs.
2. 2026-09-22 menambah I0335; request reference bukan consent/delivery completed.
3. Stage/owner/ACV/outlets historical null; snapshot 2026-10-01 menyediakan nilai sumber dan evidence.
4. Scope kosong/unauthorized/future/focus invalid ditolak, graph endpoints dan references resolve.
5. Seed rerun tidak menggandakan IDs; cancellation/driver failure bukan empty success.
6. Full existing tests, go vet, race check tetap lulus. Live Aura checks dilabeli belum dijalankan tanpa instance/secret; no deployment/benchmark claim sebelum diukur.

Handoff kepada O2: constructor/config adapter, schema+seed+manifest, cara setup satu kali, source-grounded query examples dan pass/fail nyata. Tidak menyentuh routes/controller/service/wiring mereka.

## Persetujuan dan gerbang berikutnya

User sudah menyetujui engine, desain dan plan Native. Implementasi di branch feat/neo4j-context-o1; belum commit/push/merge, izin publikasi PR sebelumnya bukan izin otomatis publikasi tahap baru. Credentials diperlukan hanya untuk load/live acceptance, bukan untuk tests/persiapan offline. Batas/deviasi lokal dicatat dalam runbook dan ledger; approval kontrak consumer O2 tetap terpisah.

Primary references: [Neo4j Go connection](https://neo4j.com/docs/go-manual/current/connect/), [managed transactions](https://neo4j.com/docs/go-manual/current/transactions/), [schema constraints](https://neo4j.com/docs/cypher-manual/current/schema/constraints/create-constraints/). Project authority: data-repository-contract-draft.md, response-o2.md, source-mapping.md, READMEHack.md dan arahan database statis terbaru user.
