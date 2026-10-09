# Relio — Jawaban review kontrak data dari O2

**Tanggal:** 9 Oktober 2026  
**Versi jawaban:** `o2-contract-review-v0.1`  
**Reviewer:** O2 — aplikasi backend  
**Status:** jawaban review O2 untuk dikonfirmasi O1; bukan laporan implementasi selesai atau persetujuan bersama yang sudah tercatat.

Acuan: [draft kontrak O1](data-repository-contract-draft.md), [rencana pengembangan](development-plan.md), [arsitektur](architecture.md), dan [README backend](../README.md). Dokumen ini melengkapi draft O1, tidak menimpa draft, arsitektur, atau daftar pertanyaan bersama.

## 1. Baseline dan perkembangan terbaru

- Dataset FIX; database bisnis STATIS dan disiapkan sekali. Tidak ada ETL otomatis, connector, scheduler, refresh dataset, atau parsing CSV pada request/server startup.
- Satu aplikasi Go dengan `main.go`, `routes/`, `controllers/`, `services/`, `repository/`, dan `models/` langsung di bawah `backend/`. Tidak menambah `internal/`, `cmd/`, atau microservice hanya untuk membagi dua developer.
- O1 memiliki source mapping, identity/temporal rules, graph/source domain types, query, dan adapter database. O2 memiliki HTTP DTO, routes/controllers, application services, access scope, policy, scoring, JEV, comparison, cache, Copilot, dan wiring.
- Unknown/null/ambiguous tidak diubah menjadi zero/false/rejected. Semua claims penting memiliki provenance. Preseden tidak memberikan approval atau consent untuk deal aktif.
- Tidak ada approval baru, commercial writes, atau eksekusi tindakan otomatis.

### 1.1 Main terbaru yang diperiksa

`origin/main` pada commit `7d9ad30` sudah memuat merge PR #1 dari `handoff/1`. Dibanding baseline sebelumnya, perubahan O1 menambah tujuh artefak:

- `backend/docs/dataset-audit.md`.
- `backend/docs/source-audit-snapshot.json`.
- `backend/docs/source-mapping.md`.
- `backend/docs/handoff-o1-2026-10-09-8fd41a/README.md`.
- `backend/docs/handoff-o1-2026-10-09-8fd41a/source-cases-draft.json`.
- `backend/docs/handoff-o1-2026-10-09-8fd41a/evaluation-tasks-draft.md`.
- `backend/docs/handoff-o1-2026-10-09-8fd41a/evaluation-answer-key-draft.md`.

Audit/mapping/handoff README telah dibaca dari remote-tracking commit tersebut. Menurut artefaknya, O1 sudah menyediakan source observations, hashes/counts, mapping, slice P04, dan 15 kasus sumber untuk review. Observations ini tidak diverifikasi ulang terhadap seluruh dataset pada review O2 ini.

**Belum dibawa oleh merge tersebut:** source Go baru, shared domain/interface, schema database, adapter, migration, atau hasil acceptance adapter. Paket tetap DRAFT dan menunggu keputusan kontrak. JSON audit/handoff bukan API response bisnis atau database siap pakai.

Pemeriksaan memakai `fetch` dan pembacaan commit; perubahan main terbaru belum digabungkan ke working branch O2 pada saat dokumen ini dibuat.

## 2. Ringkasan jawaban F01–F08

| ID | Jawaban O2 | Hal yang masih harus dibuktikan/dikonfirmasi |
|---|---|---|
| F01 | Satu database graph; arah yang direkomendasikan Neo4j, Aura jika instance tersedia. Database bisnis read-only setelah persiapan. | Engine/deployment aktual, instance, credential, schema/data siap, dan pengujian koneksi. |
| F02 | Tambah `DealFactsRepository` dan nullable facts; pertahankan kontrak legacy selama transisi. O2 memigrasikan consumer secara eksplisit. | Ack O1 atas domain/interface dan fixtures sebelum migrasi consumer. |
| F03 | Date-only, seluruh hari Asia/Jakarta, cutoff 2026-10-01, interval internal half-open. `selesai` terisi diperlakukan sebagai hari terakhir menjabat berdasarkan konvensi tim. | Ack boundary dan tests; konvensi bukan bukti bahwa makna end-day sumber telah diverifikasi. |
| F04 | Account-only tetap account-scoped; tidak memakai single-deal fallback pada MVP awal; ambiguity dipertahankan. | Reviewed linkage/identity rules dan registry O1. |
| F05 | Scope demo terbatas dan dibentuk server; izin analog terpisah; tidak membangun production RBAC sebagai prasyarat. | Gate/login deployment dan daftar akun analog/company sources yang diizinkan. |
| F06 | Template/candidate wajib occurrence dan relevance reason berbukti; policy facts per request. Unknown izin boleh dipertimbangkan dengan warning, bukan diklaim executable. | Taxonomy/relevance registry dari sumber, ack rules, dan policy fixtures. |
| F07 | Depth 1/default dan 2/max, 150 nodes, 300 edges, page 50; selected bundle lengkap atau error. | Evidence cap berdasarkan pengukuran, bukan angka rekaan. |
| F08 | Terima Fact/meta/context/truncation/typed errors dan selected batch atomik; DTO/HTTP mapping dimiliki O2. | Final Go types, public DTO/error codes, contract version, dan serialization tests. |

## F01 — Database dan penyerahan data statis

**Jawaban:** terima satu database graph; pilihan deployment masih bersyarat.

1. Arah yang disarankan ialah Neo4j, menggunakan Neo4j Aura jika instance tersedia untuk tim. Jangan menambah Supabase/MongoDB sebagai database kedua tanpa kebutuhan konkret. Pilihan vendor bukan bukti koneksi atau data siap.
2. O1 menyiapkan schema/data bisnis sekali di awal, berikut canonical ID registry, source/evidence mapping, dan manifest versi/checksum. Hashes audit yang sudah tersedia adalah input evidence, bukan otomatis final published `dataset_version`.
3. Runtime backend hanya membaca melalui adapter. Tidak ada import/penarikan dataset pada request atau restart API.
4. O1 menyerahkan constructor/config adapter, kebutuhan koneksi, readiness behavior, dan cara menjalankan integration tests. Secret melalui environment/kanal aman; tidak dimasukkan ke Git, dokumentasi, browser, atau health response.
5. Gunakan credential read-only untuk runtime jika engine mendukung. Jika DB/schema/data belum siap, kembalikan `data_not_ready`, bukan empty success atau inisialisasi diam-diam.
6. Sebelum instance dikonfirmasi, logical schema, domain types, interface, registry rules, dan test-only fixtures tetap bisa dikerjakan.

## F02 — Migrasi repository dan nullable facts

**Jawaban:** tambah `DealFactsRepository`; jangan memaksakan rich facts ke model lama.

- `DealRepository` dan `models.Deal` existing dipertahankan selama transisi. Perubahan signature/shared files harus melalui koordinasi.
- O1 menyediakan domain types dan interface tambahan di `models/` dan `repository/`. O2 memiliki perubahan consumer service/controller/HTTP DTO dan `models/deal.go` existing jika diperlukan.
- Gunakan `Fact[T]` dengan value nullable, state, temporal basis, evidence IDs, dan limitations. `known` memerlukan value/bukti; unknown atau ambiguous tidak memilih nilai default.
- Tidak membuat compatibility bridge yang mengubah unknown menjadi zero atau membuang provenance.
- O2 memigrasikan dua endpoint deal beserta tests secara eksplisit setelah kontrak minimum dan shared types di-ack. Route tetap `/api/deals` dan `/api/deals/{deal_id}`; perubahan response didokumentasikan sebelum frontend menggunakan kontrak final.
- Bentuk model/response scaffold saat ini bukan DTO facts final dan tidak menjanjikan dukungan historical.

## F03 — Tanggal, timezone, dan employment boundary

**Jawaban:** gunakan date-only dan seluruh hari `Asia/Jakarta`.

1. `as_of` berbentuk `YYYY-MM-DD`. Jika absent, default `2026-10-01`; empty, duplicate, invalid, atau setelah cutoff ditolak tanpa clamp.
2. Event date harus `<= as_of`. Jika timestamp asli tersedia, batas query adalah awal hari berikutnya secara eksklusif dalam Asia/Jakarta, bukan midnight awal tanggal terpilih.
3. Interval internal `[valid_from, valid_to)`. Untuk kontrak implementasi, `selesai` terisi disepakati sebagai hari terakhir menjabat, lalu dinormalisasi +1 hari untuk `valid_to`. Raw source date tetap tersimpan. Ini konvensi tim yang perlu ack O1, bukan hasil verifikasi end-day semantics sumber.
4. Interval overlap yang ditemukan audit dipertahankan dan dilabeli; jangan auto-repair, truncate, atau memilih satu jabatan tanpa bukti. Cases/tests perlu mempertimbangkan boundary yang disepakati.
5. Sebelum creation date, detail menghasilkan `not_visible_at_snapshot` dan list mengecualikan deal tersebut.
6. Tanpa source timestamp jangan membuat jam kejadian, `recorded_at`, atau `observed_at` palsu. Baseline rekonstruksi event-time, bukan klaim knowledge-time/bitemporal audit.
7. Historical projection menyaring future fields, quotes, links, approval/consent/outcome, dan validity end; current snapshot facts tanpa history tetap null/snapshot-only.
8. Context/date/version/access yang sama berlaku untuk seluruh panel dan bundles; cursor tidak boleh dipakai pada context/query berbeda.

Controller saat ini hanya menerima snapshot 2026-10-01 dan meneruskan token midnight UTC. O2 mengganti behavior sementara ini setelah kontrak temporal diterima dan tests boundary tersedia; behavior date-only seluruh hari belum dianggap diimplementasikan sekarang.

## F04 — Account-only interaction dan identity resolution

**Jawaban:** aturan konservatif; tidak menggunakan single-deal fallback pada MVP awal.

- Interaction tanpa native `deal_id` tetap account-scoped. Boleh tampil sebagai konteks akun dengan label jelas, bukan otomatis fakta deal tertentu.
- Link ke deal hanya melalui explicit reference atau reviewed linkage rule yang memiliki bukti, metode, dan versi. Shared account saja tidak menjadi native interaction→deal edge.
- Nama/email similarity tidak cukup untuk merge person. Identitas ambiguous memiliki chosen `node_id: null`, candidate refs yang authorized, match method, dan evidence/limitations.
- Exact current-email match tidak otomatis membuktikan email/role sepanjang history. Alamat historis unmatched yang dilaporkan audit tidak dibuat sebagai verified alias tanpa review/bukti.
- Organisasi luar tanpa `account_id` memakai locator organisasi bersumber, bukan account bisnis buatan.
- O1 menjadi satu pemilik identity/linkage mapping. O2 tidak parsing CSV atau membuat graph/source mapping kedua.

## F05 — Scope akses demo dan preseden analog

**Jawaban:** gunakan scope demo kecil yang dikendalikan server.

- Usulan MVP private: Sales/VP dapat membaca lima deal demo beserta akun terkait, tanpa production owner-based RBAC sebagai prasyarat. Ini baseline scope demo, bukan sistem auth produksi yang sudah dibuat.
- Persona switcher bukan otorisasi. Gate/session/token yang diperlukan deployment diverifikasi O2 di server; employee dataset bukan otomatis akun login.
- O2 membentuk `AccessScope` dari otorisasi server. Browser tidak menentukan allowlist; scope kosong berarti tidak ada izin.
- Izin analog accounts diberikan eksplisit dan terpisah dari current-deal permission. Hanya akun pembanding yang diperlukan/diizinkan; tidak membuka semua sumber demi comparison.
- Company-wide evidence membutuhkan klasifikasi sumber dan izin eksplisit. Unknown scope tidak dipromosikan menjadi company-wide.
- Denominator ACV menggunakan universe prospek aktif pada snapshot sesuai rubrik, bukan filter/pagination UI. Jika akses/fakta untuk universe itu tidak cukup, laporkan limitation/unknown; jangan menghitung denominator subset diam-diam.
- O1 menerapkan scope pada hasil maupun refs/evidence. O2 menangani server authorization, redaksi HTTP, gate demo, dan paid-model protection.

Pilihan provider login bukan prerequisite untuk O1 menulis repository yang menerima scope dan fixtures allow/deny.

## F06 — Taxonomy, relevance, approval, dan consent

**Jawaban:** kandidat harus berbukti, relevance dapat dijelaskan, dan izin terpisah dari suitability.

1. Template hanya ada bila memiliki minimal satu occurrence berbukti. Candidate memerlukan minimal satu relevance reason dengan bukti deal/context dan precedent yang visible/authorized.
2. Mulai dengan deterministic/reviewed rules atas gate/request/target/konteks yang known. O1 menyerahkan registry taxonomy/relevance dengan versi; semantic reranking bukan prerequisite baseline.
3. Kategori menu tidak dibuat jika sumber tidak mendukungnya. Buyer meminta referensi bukan bukti Sales telah memberikan referensi, completed action, atau consent.
4. Raw status dipertahankan; requested/proposed/offered/approved/rejected/applied/completed/pending tidak disamakan. Outcome unknown tidak menjadi failed/success, dan Won setelah action bukan causal evidence.
5. `PolicyFacts` melekat pada occurrence/request tertentu. O2 membutuhkan requested/approved discount BPS, request/decision/application links, approver beserta bukti role, dan consent status/target/scope dengan provenance.
6. Approval 10% tidak diwariskan ke request 14%; >1000 BPS (>10%) memicu kebutuhan approval VP Sales. Role historical yang tidak didukung tetap limitation, bukan verified VP.
7. Approval/consent akun analog tidak diwariskan ke deal/target aktif. Tanpa bukti valid, izin unknown, bukan rejected atau approved.
8. Kandidat yang berbukti/relevan dengan izin unknown boleh dibandingkan secara analitis dengan warning/prasyarat seperti `requires_validation`; tidak diklaim authorized/executable. Penolakan eksplisit tetap ditampilkan sebagai kendala, bukan disembunyikan oleh score tinggi.
9. Candidate 0/1 dikembalikan apa adanya; O2 tidak menambah opsi agar compare berjalan. Penolakan satu request tidak membuat prioritas seluruh deal otomatis nol.

O1 memiliki fakta, retrieval, evidence dan relevance registry. O2 memiliki policy flags/compare validation, scoring/JEV/Choice, dan HTTP. Tidak ada commercial execution/approval baru.

## F07 — Bounds, focus, pagination, dan evidence budget

**Jawaban:** terima graph/paging bounds; evidence cap menunggu pengukuran.

| Parameter | Batas awal |
|---|---:|
| Graph depth default | 1 |
| Graph depth maksimum | 2 |
| Graph nodes maksimum | 150 |
| Graph edges maksimum | 300 |
| List/timeline/candidates page default dan maksimum | 50 |
| Action IDs untuk comparison | 2–4 unik, divalidasi O2 |

- Bounds/filter invalid ditolak, bukan diperluas diam-diam. Sort/cursor stabil dan terikat snapshot/filter/scope.
- Focus event harus terkait context deal/account yang sah, visible pada cutoff, dan authorized. Focus tidak memperluas waktu/akses.
- Seluruh edge endpoints ada pada graph hasil; keluarkan edge jika kedua endpoints tidak bisa dipenuhi dan nyatakan truncation.
- Timeline boleh lebih luas dari graph terbatas. Event/refs tetap dapat di-resolve melalui focus/filter dalam context yang sama.
- Truncation dan reasons wajib eksplisit; halaman pertama tidak menjadi bukti total hanya 0/1 candidate.
- Selected lookup satu ID boleh untuk inspeksi; O2 tetap memerlukan 2–4 unik untuk compare. Semua evidence refs selected bundle dipenuhi lengkap atau menghasilkan `bundle_limit_exceeded`.
- Angka cap evidence belum dibekukan. O1 melaporkan count/ukuran dari slice nyata dan bundle empat opsi valid jika tersedia; kalau tidak tersedia, test-only fixture diberi label. O1/O2 menetapkan cap konfigurasi berdasarkan pengukuran sebelum integrasi model dirilis. Tidak menambah action fiktif atau memotong evidence diam-diam untuk memenuhi budget.

## F08 — Metadata, DTO, typed errors, dan atomic selection

**Jawaban:** terima struktur domain; public DTO/error mapping dimiliki O2.

- Terima per-field Fact, dataset/contract versions, shared context ID, unknowns/limitations, dan explicit truncation. `context_id` dihitung/validasi server, bukan authorization token.
- Empty arrays berupa `[]`; unknown values berupa null dengan state. Unavailable/query failure bukan empty success.
- Usulan DTO target: list `{meta,bounds,items}`, detail `{meta,deal}`, graph `{meta,bounds,nodes,edges}`, timeline `{meta,bounds,events}`, evidence `{meta,evidence}`. DTO/tag final dibekukan dalam consumer contract sebelum frontend integration.
- HTTP error shape tetap `{code,message}` dengan `issues` opsional untuk batch. Existing public codes scaffold dipertahankan selama transisi atau dimigrasikan eksplisit bersama tests/documentation; repository code tidak otomatis sama dengan public HTTP code.
- Typed repository errors mempertahankan Cause/Unwrap agar context cancellation/deadline tetap dapat diperiksa. Driver/query/credential/internal cause tidak diserialisasi ke frontend atau log yang tidak aman.

| Repository category | HTTP mapping usulan |
|---|---:|
| `not_found`, `not_visible_at_snapshot` | 404; generic resource-not-found bila perlu menghindari leakage |
| `invalid_snapshot`, `invalid_query` | 400 |
| `data_not_ready` | 503 |
| `query_failed` | 500; cancellation/deadline ditangani sesuai cause oleh O2 |
| `ambiguous_reference` bila operasi membutuhkan satu referensi pasti | 422 |
| `context_mismatch` | 409 |
| `invalid_selection`, `bundle_limit_exceeded` | 422 |
| `access_denied` | 403 untuk larangan akses umum, atau generic 404 untuk resource lookup |

- Syntax/body atau jumlah ID compare bukan 2–4 ditolak O2 sebagai 400. Invalid selected entity/relevance/context lookup mengikuti kategori repository.
- Selected batch atomik: tidak mengembalikan subset sukses sebagai comparison lengkap, menghapus duplicate diam-diam, atau mengganti ID. Issues hanya memuat detail yang aman/authorized.
- Ambiguous participant yang masih dapat direpresentasikan di graph/timeline masuk result/unknowns dengan nil error; bukan alasan menggagalkan seluruh query.
- Setelah ack O1, bekukan contract version bersama; label draft tidak diubah menjadi final hanya karena dokumen ini dibuat/di-merge.

## 3. Kondisi kode O2 sekarang

Fondasi pada branch `feature/deal-api-foundation`:

- GET `/healthz`, `/api/deals`, dan `/api/deals/{deal_id}` tersedia.
- Controller/service/routes beserta unit tests sudah ada; stdlib `net/http` tetap dipakai.
- Runtime `main.go` memakai repository nil dan menghasilkan 503 untuk request deal valid. Tidak ada data fixture yang disajikan sebagai data bisnis.
- `models.Deal`, `DealRepository`, dan fixed-snapshot validation masih kontrak sementara; belum facts nullable/temporal/access rich contract.
- Graph, timeline, evidence, assessment, actions, dan Copilot belum diimplementasikan pada fondasi ini.
- Pemeriksaan pada sesi review ini: `go test ./...` lulus dan `go vet ./...` lulus pada kode working branch. Ini bukan acceptance database, source checksum verification, JEV live, atau UI E2E.

Pada snapshot Git yang diperiksa, main terbaru belum memiliki fondasi endpoint O2. Merge O1 terbaru hanya menambah artefak docs/audit/handoff, bukan adapter yang dapat langsung dipasang ke service.

## 4. Urutan kerja tercepat setelah merge O1

### 4.1 Pilihan integrasi

**Rekomendasi: integrasikan jawaban O2 bersama perubahan foundation yang sudah selesai dan lulus tests; tidak menunggu seluruh pekerjaan setelah kontrak selesai.**

| Pilihan | Penilaian |
|---|---|
| Merge jawaban saja | Mengurangi waktu komunikasi, tetapi O1 masih melihat main dengan scaffold API lama. Cocok bila ada aturan branch khusus, bukan pilihan default untuk proyek ini. |
| Merge jawaban + perubahan foundation hingga saat ini | Pilihan utama: O1 melihat keputusan review dan consumer aktual. Foundation unavailable tetap diberi label jujur; merge bukan klaim MVP/data siap. |
| Bangun seluruh langkah yang tertunda sebelum mengirim jawaban | Tidak disarankan: menunda ack O1 dan berisiko membuat model/interface/query yang tumpang tindih. Implementasi consumer yang tergantung kontrak menunggu ack minimum, bukan menunggu semua graph selesai. |

Langkah 1 yang tertunda adalah **kesepakatan kontrak**, bukan membangun seluruh graph atau menulis ulang audit. Jawaban ini menyelesaikan sisi review O2; masih diperlukan ack/koreksi O1 dan deklarasi Go bersama sebelum migrasi consumer.

### 4.2 Urutan praktis

1. O2 memasukkan perubahan main terbaru ke working branch sebelum mengajukan integrasi balik ke main; review perubahan, rerun tests, dan pastikan tidak menimpa artefak O1.
2. Simpan jawaban ini sebagai commit dokumentasi terpisah dari commit endpoint yang sudah ada, lalu ajukan PR foundation + jawaban ke main. Jangan mengikutsertakan secret/dataset yang seharusnya di-ignore. Merge/push dilakukan hanya setelah diotorisasi.
3. O1 membaca main terbaru dan memberi ack/koreksi per F-ID, terutama F02/F03/F04/F05/F08. Catat contract version, ownership file, dan kesepakatan; F01 koneksi nyata dan F07 evidence cap tetap memiliki gerbang pembuktian terpisah.
4. O1 segera menyerahkan shared domain types, interfaces, typed errors, dan test-only fixtures dalam perubahan kecil. O2 review sebelum menggunakan types tersebut; tidak menulis registry/identity/query alternatif.
5. Setelah shared Go contract tersedia, jalankan pekerjaan paralel dengan write set terpisah:
   - **O1:** schema/data statis, adapter read-only, graph/timeline/evidence, identity/source references, dan adapter acceptance tests.
   - **O2:** nullable DTO, migrasi deal service/controller, temporal/access validation, error mapping, routes context, dan HTTP/service tests dengan fixtures test-only.
6. Integrasikan satu vertical slice nyata sebelum memperlebar cakupan. Mulai dari **DL-004/P04**, karena O1 sudah menyediakan mapping dan date-filter observations. Jangan menunggu seluruh lima deals atau dua action candidates untuk membuktikan graph/timeline/evidence.
7. Lanjutkan assessment/action comparison/Copilot setelah context/evidence boundary dapat dipercaya. POC koneksi live JEV bisa berjalan paralel jika credential tersedia, tetapi test input bukan acceptance MVP dan model output bukan source truth.

### 4.3 Acceptance slice pertama P04

Gunakan observations O1 sebagai bahan expected assertions setelah checksum sumber diverifikasi:

- Deal ID `DL-004` berbeda dari account ID `P04`.
- Pada 2026-09-01, account-context interactions `I0284` dan `I0314` visible; `I0335` belum visible.
- Pada 2026-09-22, `I0335` ikut visible. Quote kebutuhan referensi tidak bocor pada cutoff sebelumnya.
- Ketiga records tetap account-scoped, bukan native deal edges buatan.
- Historical stage/owner/ACV/outlet tidak diisi dari snapshot tanpa temporal support.
- Graph/timeline/evidence mempunyai context sama dan refs yang valid. Reference request tidak menjadi consent atau completed delivery.
- Source-audit/handoff JSON tidak menjadi API business response. Runtime data berasal dari query adapter database yang sudah siap.

Paket kasus O1 mempercepat expected assertions; tidak menggantikan contract approval, adapter tests, atau bukti koneksi/data ready.

## 5. Batas review ini

Dokumen ini hanya menambahkan jawaban review dan urutan integrasi. Draft O1 dan dokumen bersama tidak diubah. Tidak membangun source adapter, memilih instance yang belum tersedia, menyiapkan DB, menjalankan ETL, atau melakukan commit/push/merge working branch sebagai bagian penulisan dokumen.
