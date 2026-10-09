# Relio — Draft kontrak data dan repository O1 ↔ O2

**DRAFT — menunggu persetujuan Orang 2**  
**Tanggal:** 9 Oktober 2026 · **Versi usulan:** `context-contract-v0.1-draft`  
**Penulis:** Orang 1 / context graph dan data. Semua nama tipe, method, enum baru, batas query, dan aturan normalisasi di sini adalah usulan. Dokumen ini bukan persetujuan bersama, implementasi adapter, hasil query database, atau laporan acceptance lulus.

## 0. Acuan, arahan terbaru, dan bukti yang diperiksa

- [Development plan](development-plan.md), khususnya H01–H04, pembagian file, dan contract tests.
- [PRD Final MVP v1.0](../../../Relio_PRD_Final_MVP_v1.0_Agent_Ready_ID.md), khususnya §2–3, §5, §7–8 dan §12.
- [READMEHack.md](../../../READMEHack.md), kamus kolom dan hubungan antar-file.
- Header CSV di `../../../Datasets/`, bentuk dua record pertama `interactions.jsonl`, dan baris deal prospek pada `crm_deals.csv`.
- Source aktual `../main.go`, `../routes/routes.go`, `../models/deal.go`, `../repository/deal_repository.go`, `../services/deal_service.go` beserta tests existing.

**Penyesuaian terbaru dari user:** dataset FIX; database bisnis STATIS, disiapkan sekali di awal. Tidak membangun pipeline ETL, connector, scheduler, refresh dataset otomatis, atau parsing CSV saat request/server startup. `as_of` memfilter riwayat yang telah disimpan. Kewajiban pipeline ETL/reimport dalam PRD dan development plan lama tidak menjadi pekerjaan implementasi pada kontrak ini. Kualitas sumber, integritas ID, provenance, dan uji temporal tetap diperlukan untuk database yang disiapkan sekali. Jika database belum siap, kembalikan status belum siap; jangan diam-diam mengisi atau mengambil dataset baru.

Engine database **belum difinalkan oleh kontrak ini**. Neo4j Aura telah dibahas sebagai calon deployment; ketersediaan instance, pilihan tim, schema, dan koneksi belum diverifikasi pada tugas ini. Interface tidak bergantung pada Aura, Docker, SQL, Cypher, atau driver tertentu.

Dataset kompetisi sendiri dinyatakan sintetis/fiktif oleh README. Istilah **data asli** di sini berarti file kompetisi yang tersedia, dibedakan dari contoh kontrak/fixture tambahan buatan developer. Pemeriksaan di atas bukan audit seluruh baris, checksum final, identity resolution, atau verifikasi query database.

Temuan struktur yang benar-benar diperiksa:

| Fakta sumber | Implikasi kontrak |
|---|---|
| `crm_deals.csv` memiliki pasangan `DL-001/P01` sampai `DL-005/P05` | `deal_id` dan `account_id` wajib dibedakan; mapping ini diperiksa dari baris CSV, bukan hanya contoh PRD. |
| `interactions.jsonl` memiliki `account_id`, tidak memiliki `deal_id` | Account-only interaction tidak otomatis menjadi fakta salah satu deal; linkage harus punya aturan/bukti dan ambiguity tetap terlihat. |
| `contact_employment_history.csv` tidak memiliki ID baris; kolomnya `contact_id,account_id,organisasi,jabatan,mulai,selesai` | Perlu locator turunan; jangan mengaku locator tersebut ID sumber native. |
| Tanggal pada sampel interaksi dan kolom CSV berupa tanggal kalender | Jangan menciptakan jam kejadian, `recorded_at`, atau urutan waktu intrahari. |
| CRM memuat `stage` dan `stage_sejak`, bukan log transisi stage lengkap | Historical stage tetap unknown tanpa bukti tambahan; `stage_sejak` tidak otomatis membuat semua nilai CRM historis. |
| `decision_log.csv` memiliki `deal_id` yang dapat kosong dan link `bukti_interaction_id`; billing memiliki `decision_id` | Approval account-only dan status kontrak perlu linkage eksplisit sebelum dianggap milik request/deal tertentu. |

### Kontrak existing yang dipertahankan

Source aktual memakai Go 1.22, stdlib `net/http`, dan hanya route `GET /healthz`. `DealService` meneruskan repository calls dan memvalidasi ID kosong; belum ada adapter database dalam scaffold yang diperiksa.

```go
// EXISTING: kutipan signature aktual, bukan perubahan yang diusulkan.
type DealRepository interface {
    List(ctx context.Context, asOf time.Time) ([]models.Deal, error)
    FindByID(ctx context.Context, dealID string, asOf time.Time) (models.Deal, error)
}
```

`models.Deal` aktual: `ID`, `AccountID`, `OwnerID`, `Stage`, `Status` bertipe `string`; `StageSince`, `CreatedAt` bertipe `time.Time`; `OutletCount int`; `PotentialACVIDR int64`. Nilai ini belum bisa membedakan unknown, snapshot-only, provenance, dan angka nol yang benar-benar diketahui.

Rekomendasi: pertahankan file/signature existing; sepakati interface **tambahan** `DealFactsRepository` untuk context kaya sebelum O2 mengubah consumer. Tidak mengisi unknown dengan zero value agar cocok dengan `models.Deal`. Legacy `DealRepository` belum dapat menjanjikan seluruh semantik draft ini. Jembatan legacy, perubahan model, atau migrasi consumer harus diputuskan eksplisit oleh O2; source existing tidak diubah di tugas ini.

## 1. Batas tanggung jawab dan aturan bersama

O1 menyediakan schema/data yang disiapkan sekali, identitas temporal, provenance, query graph/timeline/evidence/preseden, dan adapter database setelah interface disepakati. O2 memiliki routes/controllers, DTO HTTP, wiring, akses, aplikasi policy, scoring, JEV, comparison, cache, dan Copilot. O2 mengonsumsi query O1, tidak membuat mapping CSV/query graph alternatif. Pembagian ini dapat berjalan di satu proses Go.

Database bisnis bersifat read-only setelah persiapan. Interface ini tidak memiliki method write. Cache/model metadata adalah tanggung jawab terpisah O2; hasil model tidak menjadi fakta graph tanpa proses verifikasi terpisah yang disepakati. Tidak ada approval diskon baru atau eksekusi komersial otomatis.

### 1.1 Snapshot, waktu, dan akses

1. Cutoff maksimum tetap **2026-10-01**. Draft mengusulkan API date-only `YYYY-MM-DD`, zona kalender `Asia/Jakarta`, seluruh hari terpilih termasuk dalam query. Event date harus `<= as_of`. Jika sumber benar-benar mempunyai timestamp, gunakan batas awal hari berikutnya secara eksklusif dalam zona yang disepakati; jangan memakai pukul 00:00 tanggal terpilih sebagai akhir hari. Zona/boundary ini menunggu O2.
2. `as_of` di atas cutoff, tanggal tidak valid, snapshot version salah, atau konteks kosong ditolak; tidak silently clamp ke cutoff. Tanggal sebelum suatu deal dibuat menghasilkan `not_visible_at_snapshot` untuk detail dan mengecualikan deal dari list. Tanggal sebelum riwayat tersedia tidak dianggap query DB gagal.
3. `event_at` adalah tanggal kejadian; `source_timestamp` hanya terisi jika sumber menyimpan jam/offset. `recorded_at`/`observed_at` nullable dan hanya terisi bila tersedia native. Waktu persiapan database bukan waktu bisnis diketahui. Rekonstruksi event-time bukan bukti rekonstruksi knowledge-time.
4. Usulan interval internal `[valid_from, valid_to)`; `valid_to = null` berarti ujung belum diketahui atau masih terbuka, dibedakan lewat metadata. Untuk employment, README menyebut `selesai` kosong masih menjabat. Makna `selesai` terisi (hari terakhir vs hari pertama tidak aktif) belum pasti: rekomendasi hari terakhir inklusif, dinormalisasi +1 hari hanya setelah disepakati. Tanggal mentah tetap di provenance.
5. Fakta snapshot-only sebelum 2026-10-01 memakai `value: null`, `state: snapshot_only`, dan limitation. Evidence snapshot-only juga tidak dibuka sebagai bukti historis. Nilai akhir validity, outcome, resolved status, label/nama/email saat ini, atau kutipan yang baru tersedia setelah cutoff ikut disaring; memfilter event saja belum cukup. Employment yang berakhir setelah `as_of` tidak membocorkan tanggal akhir future dalam projection historical.
6. Snapshot memuat versi dataset tetap dan versi kontrak; `context_id` dihitung server dari versi, tanggal, zona, serta scope akses yang telah diotorisasi. Semua panel/bundle memakai konteks identik. Cursor juga terikat query/filter/sort; beda tanggal/versi/scope tidak bisa memakai cursor lama. `context_id` bukan bearer token atau pengganti otorisasi.
7. O2 membentuk scope akses dari server, tidak menyalin allowlist dari browser. O1 menerapkan allowlist pada seluruh hasil/refs, termasuk preseden analog. Scope kosong berarti tidak ada izin. Deal-scoped evidence memerlukan deal dan account yang diizinkan; account-only evidence memerlukan account yang diizinkan dan tetap berlabel account-only. Company-wide evidence hanya boleh keluar dengan izin eksplisit; sumber tidak jelas scopenya tidak menjadi company-wide otomatis.
8. Panel timeline boleh lebih luas daripada graph yang dibatasi node. Reference tetap harus dapat di-resolve dalam konteks yang sama; status `truncated` tidak mengizinkan dangling edge. `focus_event_id` mengambil graph pendukung event yang dipilih tanpa mengubah tanggal/scope.

### 1.2 Identitas dan provenance

- ID bisnis native dipertahankan: `DL-…`, `P…/C…`, `K…`, `E…`, `I…`, `D-…`, contract/outlet/feature ID asli. ID graph diberi namespace, misalnya `deal:DL-001`, `account:P01`, `person:contact:K001`, `person:employee:E01`; source record ID tidak ditukar dengan graph ID.
- Nama/email saja tidak cukup untuk merge person. `participant` unresolved tetap memiliki locator yang stabil, raw reference yang terotorisasi, dan `candidate_node_ids`; satu candidate pun tidak menjadi verified tanpa metode/bukti yang memenuhi aturan.
- Entitas/edge/event/occurrence/evidence turunan memakai namespace + digest deterministik atas locator sumber dan claim discriminator, **bukan counter urutan query atau teks ringkasan saja**. Encoding key wajib unambiguous (misalnya canonical JSON). Berbeda actor/target/waktu/nilai/status/request bukan occurrence yang sama. Identity ambiguity bukan alasan memilih ID pertama.
- Source native: `source_record_id` persis ID sumber, `record_id_kind: native`. Sumber tanpa ID: gunakan locator `derived:<digest>` dengan `record_id_kind: derived`, `source_key` dari semua kolom pembeda atau composite key yang diverifikasi unik. Locator ini usulan ID teknis, bukan temuan ID bisnis baru. Konflik composite key/duplicate non-identik dilaporkan, tidak diberi nomor acak.
- Evidence adalah satu claim/field/span tertentu. `source_field` harus nama kolom/key aktual; span Unicode-code-point `[start,end)` terhadap field asli bila relevan, disertai checksum field. CSV scalar tidak memerlukan span. Untuk sumber tanpa ID, locator dan checksum file memungkinkan penelusuran ke row tetap. Jangan memasukkan seluruh thread ke satu evidence.
- Key lengkap untuk resolver disimpan internal; `source_key` yang dikirim hanya berisi bagian key yang visible/authorized. Jangan membocorkan `selesai` future, seluruh isi row, atau metadata snapshot-only melalui composite key. Jika key tidak aman untuk ditampilkan, map dapat kosong dan resolver memakai opaque `source_record_id` + checksum; limitation menjelaskan key tidak ditampilkan tanpa menyebut nilai tersembunyi.
- Kutipan, actor, nilai, scope, status, atau relasi penting wajib ditopang evidence. Sumber agregasi memiliki daftar locator/manifest membership lengkap meski excerpt pendek. Digest/manifest/source mapping disimpan saat database disiapkan; aplikasi tidak membaca CSV untuk merekonstruksinya saat request.

## A. Matriks kontrak data

Notasi Go: `*T` nullable; `[]T` required array, empty berupa `[]`, bukan `null`. `Fact[T]` menyimpan `value *T`, `state`, `temporal_basis`, `evidence_ids`, dan `limitations`. `state`: `known`, `unknown`, `ambiguous`, `snapshot_only`; `temporal_basis`: `event`, `interval`, `snapshot`, `undated`. State/basis wajib walau value null. `known` harus memiliki value dan bukti; `unknown/ambiguous` tidak memilih satu nilai; historical snapshot-only tidak membawa nilai future. String kosong bukan representasi unknown.

`Date` adalah string kalender valid `YYYY-MM-DD`. Referensi kosong hanya di array; field ID yang tidak diketahui menggunakan pointer/null atau `ParticipantRef`. Semua enum adalah daftar usulan; raw status/tipe sumber tetap disimpan untuk audit. Untuk tiap field, kolom terakhir mencakup contoh pemakaian dan kondisi unknown.

### A1. Envelope dan status dataset

| Field / tipe | Required / nullable | Makna | Sumber/provenance | Temporal | Pemakaian / unknown |
|---|---|---|---|---|---|
| `contract_version string` | Required | Versi schema consumer | Draft disepakati O1/O2 | Tetap per kontrak | `context-contract-v0.1-draft`; tidak mengaku final. |
| `dataset_version string` | Required | Identitas data bisnis tetap | Manifest file/checksum + schema/mapping revision persiapan | Tetap selama demo | Belum ada manifest final → `data_not_ready`, bukan versi rekaan. |
| `as_of Date`, `max_as_of Date`, `calendar_zone string` | Required | Cutoff efektif dan batas dataset | Request tervalidasi + konfigurasi bersama | Semua query sama | Maksimum `2026-10-01`; lebih besar ditolak. |
| `context_id string` | Required | Korelasi panel/bundle | Derivasi server, termasuk access scope | Berubah jika tanggal/scope/versi berubah | O2 menolak response context lama di UI/cache. |
| `data_state string` | Required | `ready/partial/not_ready` | Manifest hasil persiapan + readiness adapter | Bukan fakta bisnis | `partial` hanya untuk limitation yang teridentifikasi; core tidak tersedia → error. |
| `limitations []string` | Required | Batas sejarah/kualitas | Source audit, proyeksi query | Tidak membocorkan detail future | `stage_history_unavailable`. |
| `unknowns []Unknown` | Required | Path, sebab, refs pendukung | Hasil source mapping/query O1 | Snapshot-safe | Ambiguous identity dapat membawa refs candidate yang diizinkan. |
| `truncated bool`, `truncation_reasons []string`, `next_cursor *string` | Required / cursor nullable | Batas hasil, bukan absence fakta | Query budget/paging | Cursor terikat konteks dan filter | Empty lengkap berbeda dari empty akibat batas. |

### A2. Deal

| Field / tipe | Required / nullable | Makna bisnis | Sumber/provenance | Temporal | Pemakaian / unknown |
|---|---|---|---|---|---|
| `deal_id string` | Required | Identitas opportunity | `crm_deals.deal_id` | Tidak berubah oleh tanggal query | `DL-001`, bukan `P01`. |
| `account_id string` | Required | Akun opportunity | `crm_deals.account_id`; FK `crm_accounts` | Structural linkage bersumber; bukan klaim semua metadata akun historis | `P01`; FK invalid membuat readiness gagal. |
| `deal_type Fact[string]` | Value nullable | Baru/renewal/ekspansi | `crm_deals.tipe` | Snapshot unless sejarah didukung | Tidak infer `baru` hanya dari prefix akun. |
| `owner_id Fact[string]` | Value nullable | Owner deal, bukan owner akun | `crm_deals.owner_id` → `employees.employee_id` | Snapshot-only jika history owner tidak tersedia | Historical null; jangan substitusi `account_owner_id`. |
| `stage Fact[string]` | Value nullable | Stage yang diketahui pada cutoff | `crm_deals.stage` atau evidence stage eksplisit | Current snapshot tanpa log → historical null | Null + `snapshot_only`; label unknown dibuat O2, bukan string stage palsu. |
| `stage_since Fact[Date]` | Value nullable | Awal stage yang terdokumentasi | `crm_deals.stage_sejak` | Tidak menjadi bukti semua stage sebelumnya | O2 hanya hitung stage age bila stage dan tanggalnya valid pada konteks. |
| `created_at Fact[Date]` | Value nullable | Tanggal deal dibuat | `crm_deals.dibuat` | Dapat dipakai sebagai event date; date-only | Missing tetap null; detail sebelum created date tidak visible. |
| `planned_outlets Fact[int]` | Value nullable | Outlet rencana deal; bukan outlet aktif prospek | `crm_deals.outlet`; cek konflik `crm_accounts.jumlah_outlet` | Snapshot-only tanpa perubahan terdokumentasi | Nilai 0 known ≠ null; konflik tidak diatasi dengan maksimum. |
| `potential_acv_idr Fact[int64]` | Value nullable | Potensi nilai tahunan rupiah, bukan revenue/margin | `crm_deals.nilai_tahunan` | Snapshot-only kecuali ada bukti nilai historis | Integer rupiah; missing ≠ `0`; tidak dihitung dari billing akun lain. |
| `status Fact[string]` | Value nullable | Terbuka/Won/Lost menurut sumber | `crm_deals.status` | Snapshot-only kecuali lifecycle terdokumentasi | Lost snapshot tidak retroaktif membuat deal lampau Lost. |
| `record_evidence_ids []string` | Required | Bukti structural ID/account linkage | Field evidence row CRM | Hanya refs visible pada snapshot; historical refs dapat kosong | Semua critical scalar facts memiliki evidence sendiri dalam `Fact`. |

Konflik dua nilai untuk fakta yang sama: value null + `ambiguous`, source alternatives melalui evidence/unknowns; jangan diam-diam memilih salah satu. ACV denominator/coverage/scoring bukan keluaran repository ini. O2 perlu mendapatkan facts prospek aktif pada snapshot penuh yang sesuai aturan PRD, bukan menghitung denominator dari filter halaman; akses input tersebut perlu disepakati di F.

### A3. Graph dan participant identity

| Field / tipe | Required / nullable | Makna | Sumber/provenance | Temporal | Pemakaian / unknown |
|---|---|---|---|---|---|
| `node_id string`, `node_type string`, `entity_id string` | Required | ID graph, tipe, ID entitas/locator | Source native/derived registry | Stabil lintas query dan budget | `node_type` tidak berubah menjadi tipe driver DB. |
| `person_kind *string`, `person *ParticipantRef` | Nullable | Employee/contact/unresolved participant | Employee/contact IDs; email + history bersumber | Match pada event date | Ambiguous candidate tidak di-merge ke verified person. |
| `label Fact[string]` | Value nullable | Nama/judul yang dapat dibuktikan | Field `nama/subjek` aktual | Current label tidak bocor di historical | Fallback UI boleh menampilkan ID, bukan mengarang nama lama. |
| `edge_id`, `edge_type`, `source`, `target string` | Required | Relasi bertipe, ujung berupa node IDs | Native FK atau claim + evidence locator | Edge bersumber pada cutoff | Kedua ujung harus ada di graph hasil; tidak boleh dangling. |
| `valid_from *Date`, `valid_to *Date`, `open_end_reason *string` | Nullable | Masa berlaku relasi | History `mulai/selesai`, tanggal event, atau unknown | Interval draft §1.1; mask ujung future | Null start ≠ sejak awal waktu; snapshot linkage berlabel snapshot. |
| `source_timestamp`, `observed_at`, `recorded_at *time.Time` | Nullable | Timestamp asli berbeda makna | Hanya jika sumber tersedia | Filter sesuai basis; tanpa knowledge-time jangan klaim mengetahuinya | Sampel sumber date-only → semua timestamp ini null. |
| `verification_state`, `match_method string` | Required | `verified/inferred/ambiguous` dan cara linkage | FK/exact supported identity/reviewed claim rule | Match harus valid pada waktu terkait | `SIMILAR_PRECEDENT` selalu inferred, bukan approval. |
| `evidence_ids`, `event_ids []string` | Required | Proof dan linkage timeline | Source/claim registry | Seluruh refs snapshot+scope-safe | Kosong hanya untuk container teknis yang tidak mengklaim fakta. |
| `depth`, `max_nodes`, `max_edges int`, `focus_event_id *string` | Request bounds / focus nullable | Batasi traversal dan fokus | Request tervalidasi O2, cek defensif O1 | Tidak memperluas cutoff/scope | Batas angka rekomendasi ada di F. |
| `node_id *string`, `raw_ref *string`, `candidate_node_ids []string` pada participant | Node/raw nullable; array required | Resolved vs unresolved identity | `dari/ke/peserta`, employee/contact/history | Valid pada event date | Unknown kosong; ambiguous berisi candidates tanpa chosen node. |

Node types mengikuti PRD: `Deal`, `Account`, `Person`, `Interaction`, `Event`, `Decision`, `ActionOccurrence`, `Evidence`, `Contract`, `UsageAggregate`; `Outlet` opsional. ActionTemplate adalah katalog, bukan event palsu. Edge types minimum mengikuti PRD §3.3. Endpoint constraints di antaranya: `DEAL_OF_ACCOUNT: Deal→Account`, `OWNED_BY: Deal/Account→Person(Employee)`, `WORKED_AT: Person(Contact)→Account/organisasi bersumber`, `DECIDED_BY: Decision→Person(Employee)`, `SUPPORTED_BY: claim/Event/ActionOccurrence→Evidence`. Organisasi luar dengan account_id kosong memakai locator organisasi bersumber dan status matching yang jelas, tidak menciptakan account bisnis.

`ReadGraph` memasukkan semua ujung edge atau mengeluarkan edge beserta truncation reason. History relevan boleh muncul meski tidak aktif sekarang; validity tetap diperlihatkan dan future claims disaring. Event IDs yang ada di graph bisa diambil lewat timeline dengan filter event IDs. Graph yang terpotong tidak menghapus event dari timeline lengkap.

### A4. Timeline

| Field / tipe | Required / nullable | Makna | Sumber/provenance | Temporal | Pemakaian / unknown |
|---|---|---|---|---|---|
| `event_id string`, `event_type string` | Required | Kejadian stabil dan kategori PRD | Interaction/decision/history/contract claim | Hanya kejadian bersumber `<= as_of` | Tidak membuat event setiap kategori demi menu. |
| `event_at Date`, `time_precision string`, `source_timestamp *time.Time` | Date/precision required; timestamp nullable | Tanggal dan presisi asli | `tanggal/mulai/dibuat`, sesuai sumber | Date-only tidak diberi jam fiktif | Date diketahui, jam unknown → precision `date`. |
| `status Fact[string]`, `raw_status *string` | Value/raw nullable | Status klaim pada kejadian | Quote atau keputusan sumber | Perubahan kemudian menjadi event/status evidence terpisah | `Menunggu→pending`, `Disetujui→approved`, `Ditolak→rejected`; tidak mengisi unknown sebagai rejected. |
| `actors`, `targets`, `participants []ParticipantRef` | Required arrays | Pelaku dan pihak terkait | `dari/ke/peserta`, decision actor IDs | Resolve identity saat event | Email historis mismatch tetap ambiguous/unknown. |
| `deal_ids`, `account_ids []string`, `scope_kind string` | Required | Link deal/account atau company | Native `deal_id`, account + bukti linkage | Tidak memindahkan interaksi sebelum deal dibuat tanpa bukti | Account-only tetap tanpa deal IDs. |
| `node_ids`, `edge_ids`, `evidence_ids []string` | Required | Highlight graph dan evidence | Shared registry | Seluruh refs visible/resolvable | `focus_event_id` membantu resolve graph yang sebelumnya dibatasi. |
| `summary Fact[string]` | Value nullable | Ringkasan claim bersumber | Quote/normalisasi direview, bukan hasil model otomatis | Tidak mengambil outcome future | Unknown/unclear dapat tetap tampil dengan bukti raw yang aman. |
| `verification_state string` | Required | Kualitas linkage/kategorisasi event | Evidence dan metode normalisasi yang direview | Tidak diperkuat dari knowledge setelah cutoff | Event ambigu tidak otomatis menjadi verified karena tanggalnya valid. |

Urutan usulan: `(event_at ascending, source_timestamp bila kedua record punya timestamp asli, event_id ascending)`; implementasi harus membuat total ordering dengan key presisi yang tetap untuk mixed timestamps. Rekomendasi sederhana MVP: `(event_at, event_id)` karena sumber diperiksa date-only; `event_id` tie-break hanya untuk tampilan, bukan bukti kejadian A mendahului B pada hari sama. Jika precision jam digunakan, aturan mixed-precision dibekukan dulu. Timeline pagination keyset, tidak reorder karena label/score berubah.

### A5. Evidence

| Field / tipe | Required / nullable | Makna | Sumber/provenance | Temporal | Pemakaian / unknown |
|---|---|---|---|---|---|
| `evidence_id string` | Required | Unit bukti field/claim | Native/derived locator + field/span discriminator | Stabil untuk snapshot yang sama | ID teknis evidence bukan source business ID. |
| `source_file`, `source_record_id`, `record_id_kind string` | Required | File, record, native/derived | Nama aktual file + ID atau locator tervalidasi | Tidak bergantung urutan request | History tanpa PK → derived ditandai eksplisit. |
| `source_key map[string]string`, `source_checksum string` | Required | Resolve locator dan integritas berkas tetap | PK/composite key + manifest checksum; key lengkap internal | Checksum tetap; key projection disaring sesuai snapshot/access | Key dapat `{}` jika tidak aman; checksum final belum dibuat pada draft ini. |
| `source_field string`, `span *SourceSpan` | Field required; span nullable | Field/quote offset asli | Kolom/key actual; Unicode-code-point span | Excerpt tidak melewati batas claim visible | Scalar CSV → span null. |
| `source_date *Date`, `source_timestamp *time.Time`, `temporal_basis string` | Date/timestamp nullable; basis required | Date sumber yang memang diketahui | Interaction/decision date, employment endpoint claim, snapshot annotation | Undated claim dibatasi snapshot; tidak otomatis tersedia di masa lalu | Date source null + basis snapshot dapat dipakai hanya snapshot. |
| `content_excerpt string` | Required | Nilai/kutipan tepat dan pendek | Field/span asli, redaksi O2/O1 disepakati | Satu message/claim, tidak mengandung future reply | Boleh kutipan aman; bukan seluruh `membalas_id` chain. |
| `scope_kind`, `verification_state string`; `deal_ids/account_ids []string` | Required | Scope bukti dan kualitas | Native refs + reviewed linkage | Cek cutoff AND akses | Company flag hanya untuk policy/global source yang diverifikasi. |
| `event_ids`, `edge_ids`, `occurrence_ids []string` | Required | Link ke claim/tindakan | Shared registry | Referensi future tidak dikirim | Record campuran dipecah field/span, bukan bocorkan seluruh row. |

`ReadEvidence` menerima snapshot dan scope yang sama dengan list/graph, termasuk saat dipakai untuk input JEV/Copilot. Parent thread/reply dapat ditelusuri melalui record individu; balasan sesudah cutoff tidak ada di kutipan, links, ringkasan, atau bundle. Evidence account-only tidak menjadi bukti deal-specific otomatis. Izin membaca satu deal tidak memberi izin membaca seluruh analogue accounts.

### A6. Actions, relevance, policy facts

| Field / tipe | Required / nullable | Makna | Sumber/provenance | Temporal | Pemakaian / unknown |
|---|---|---|---|---|---|
| `action_id string`, `action_type string`, `definition string`, `taxonomy_version string` | Required | ActionTemplate: opsi reusable | Occurrences + taxonomy mapping direview saat persiapan | Template historical perlu occurrence visible; definisi tidak memuat outcome future | Kategori tanpa occurrence tidak masuk katalog hasil. |
| `occurrence_id string`, `event_id string` | Required | Satu tindakan/claim historis | Claim locator interaction/decision/contract | `event_at <= as_of` | Dua waktu/target berbeda tidak digabung karena teks mirip. |
| `action_id string` pada occurrence | Required | Keanggotaan occurrence pada template | Taxonomy/claim registry yang direview | Template tidak mengubah tanggal/history occurrence | Harus sama dengan template induk pada candidate/selected bundle. |
| `actor ParticipantRef`, `targets []ParticipantRef`, `event_at Date` | Required | Pelaku, tujuan, tanggal | Quote/IDs native | Valid pada tanggal tindakan | Actor ambiguous tetap ambiguous, tidak diklaim verified. |
| `status Fact[string]`, `raw_status *string` | Value nullable | Requested/proposed/offered/approved/rejected/applied/completed/pending | Evidence claim/status asli | Perubahan stage/status action sebagai claim terpisah | Offered ≠ completed; proposed ≠ approved. |
| `outcome_observed Fact[string]`, `outcome_at *Date` | Nullable value/date | Outcome yang benar-benar tercatat | Bukti outcome terkait action | Outcome setelah cutoff tidak dipakai walau action lama visible | Null bukan failed/success; Won deal tidak otomatis outcome kausal action. |
| `account_ids`, `deal_ids`, `evidence_ids`, `node_ids`, `edge_ids []string` | Required | Scope dan highlight occurrence | Native refs + explicit supported links | Seluruh links snapshot-safe | Account-only occurrence tidak diberi deal ID tebakan. |
| `precedents []ActionPrecedent` dengan `relation string` | Required, ≥1 untuk candidate | Occurrences yang mendukung template | Occurrence valid + evidence | Setiap precedent lolos cutoff dan access | `current_deal/same_account_precedent/analog_precedent`; satu template dapat punya beberapa label. |
| `relevance_reasons []RelevanceReason`, `relevance_rule_version string` | Required, ≥1 reason untuk candidate | Mengapa cocok untuk konteks deal | Rule deterministic/reviewed match + evidence deal dan precedent | Stage snapshot-only tidak dipakai untuk historical match | Bukan skor JEV atau izin tindakan; tidak ada match → bukan kandidat sah. |
| `requested_discount_bps Fact[int32]` | Value nullable | Besar request, basis points; 1000 = 10% | Claim request; parsing exact `nilai`/quote | Tidak diwarisi dari decision lain | Unknown ≠ 0; >1000 menjadi input policy O2. |
| `request_occurrence_id *string`, `decision_id *string`, `applied_contract_id *string` | Nullable | Link request→decision→application | Decision/billing linkage + matching scope/value | Setiap link punya bukti/cutoff sendiri | Billing `decision_id` saja belum membuktikan semua request sama. |
| `approval_status Fact[string]`, `approver ParticipantRef`, `approved_discount_bps Fact[int32]` | Values nullable | Bukti keputusan dan actor/nilai terkait | `decision_log.keputusan/diputuskan_oleh/nilai`, employee role bersumber | Approval nilai/scope lama tidak berlaku untuk request baru | Absence approval → unknown, bukan rejected atau approved. |
| `consent_status Fact[string]`, `consent_target_node_id *string`, `consent_scope Fact[string]` | Values/target nullable | Consent referensi target dan cakupannya | Quote/decision consent eksplisit | Izin harus sesuai target/waktu/scope | Reference candidate ≠ consent granted. |

PolicyFacts selalu milik occurrence/request tertentu; bukan status tunggal template. Actor VP Sales harus didukung employee role yang diketahui pada waktu terkait; role snapshot-only pada historical menghasilkan limitation, bukan verifikasi palsu. O1 memasok fakta request/approval/consent serta evidence; O2 menentukan warning `requires_validation`, eligibility compare, scoring/suitability, dan HTTP response. O1 tetap menolak kandidat tanpa preseden/relevansi bersumber.

Usulan nilai `approval_status` yang diketahui: `pending/approved/rejected`; tidak ada record valid berarti value null + unknown. Usulan nilai `consent_status` yang diketahui: `granted/denied/revoked`, selalu dengan evidence target/scope; unknown tetap null. Status `applied` didukung claim kontrak terpisah dan tidak dimasukkan sebagai sinonim approved. Label feasibility `requires_validation` ditentukan O2, bukan nilai consent.

`ReadActionCandidates` mengembalikan 0/1/N yang benar-benar valid. `returned_count` adalah panjang hasil; `total_valid_count` hanya terisi bila query lengkap menghitung total. Hasil terbatas memakai `truncated: true`; jangan menyimpulkan total <2 dari halaman pertama yang terpotong. Selected lookup memvalidasi ulang existence, snapshot, scope, relevance, dan evidence, bukan hanya lookup template ID.

## B. Draft interface repository Go

**Nama/tanda tangan di bawah adalah usulan, belum ada dalam source.** Potongan pertama adalah satu unit contoh Go mandiri untuk review. Penempatan setelah disepakati: tipe domain di `backend/models/`, interface/errors di `backend/repository/`; tidak membuat `internal/` atau `cmd/`. Package contoh `contractproposal` hanya agar pembaca bisa memahami deklarasi bersama, bukan usulan package baru di repo.

```go
package contractproposal

import (
    "context"
    "time"
)

type Date string // divalidasi YYYY-MM-DD; bukan timestamp tengah malam buatan

type AccessScope struct {
    AllowedAccountIDs []string
    AllowedDealIDs []string
    AllowCompanyEvidence bool
}

type SnapshotContext struct {
    AsOf Date
    DatasetVersion string
    ContractVersion string
    CalendarZone string
    ContextID string // dihitung/validasi server, bukan otorisasi dari browser
    Access AccessScope
}

type Fact[T any] struct {
    Value *T `json:"value"`
    State string `json:"state"`
    TemporalBasis string `json:"temporal_basis"`
    EvidenceIDs []string `json:"evidence_ids"`
    Limitations []string `json:"limitations"`
}

type Unknown struct {
    Path string
    Reason string
    ReferenceIDs []string
}
type Meta struct {
    ContractVersion string
    DatasetVersion string
    AsOf Date
    MaxAsOf Date
    CalendarZone string
    ContextID string
    DataState string
    Limitations []string
    Unknowns []Unknown
}
type BoundInfo struct {
    Truncated bool
    TruncationReasons []string
    NextCursor *string
}
type DealFacts struct {
    DealID string
    AccountID string
    DealType Fact[string]
    OwnerID Fact[string]
    Stage Fact[string]
    StageSince Fact[Date]
    CreatedAt Fact[Date]
    PlannedOutlets Fact[int]
    PotentialACVIDR Fact[int64]
    Status Fact[string]
    RecordEvidenceIDs []string
}
type DealListOptions struct {
    AccountIDs []string // filter dalam scope, bukan pengganti scope
    DealTypes []string
    Limit int
    Cursor *string
}
type DealFactsPage struct { Meta Meta; Bounds BoundInfo; Items []DealFacts }
type DealFactsResult struct { Meta Meta; Deal DealFacts }

type ParticipantRef struct {
    NodeID *string
    RawRef *string
    VerificationState string
    MatchMethod string
    CandidateNodeIDs []string
    EvidenceIDs []string
}
type Validity struct {
    ValidFrom *Date
    ValidTo *Date
    OpenEndReason *string
    TemporalBasis string
}
type Node struct {
    NodeID string
    NodeType string
    EntityID string
    PersonKind *string
    Person *ParticipantRef
    Label Fact[string]
    Validity Validity
    VerificationState string
    EvidenceIDs []string
    EventIDs []string
}
type Edge struct {
    EdgeID string
    EdgeType string
    Source string
    Target string
    Validity Validity
    SourceTimestamp *time.Time
    ObservedAt *time.Time
    RecordedAt *time.Time
    VerificationState string
    MatchMethod string
    EvidenceIDs []string
    EventIDs []string
}
type GraphOptions struct {
    Depth int
    MaxNodes int
    MaxEdges int
    FocusEventID *string
}
type GraphResult struct { Meta Meta; Bounds BoundInfo; Nodes []Node; Edges []Edge }

type Event struct {
    EventID string
    VerificationState string
    EventAt Date
    TimePrecision string
    SourceTimestamp *time.Time
    EventType string
    Status Fact[string]
    RawStatus *string
    Summary Fact[string]
    Actors []ParticipantRef
    Targets []ParticipantRef
    Participants []ParticipantRef
    ScopeKind string
    DealIDs []string
    AccountIDs []string
    NodeIDs []string
    EdgeIDs []string
    EvidenceIDs []string
}
type TimelineOptions struct {
    EventIDs []string
    EventTypes []string
    ActorNodeIDs []string
    Statuses []string
    Limit int
    Cursor *string
}
type TimelineResult struct { Meta Meta; Bounds BoundInfo; Events []Event }

type SourceSpan struct {
    Start int // Unicode code points, inclusive
    End int   // Unicode code points, exclusive
    FieldChecksum string
}
type Evidence struct {
    EvidenceID string
    SourceFile string
    SourceRecordID string
    RecordIDKind string
    SourceKey map[string]string
    SourceChecksum string
    SourceField string
    Span *SourceSpan
    SourceDate *Date
    SourceTimestamp *time.Time
    TemporalBasis string
    ContentExcerpt string
    ScopeKind string
    VerificationState string
    DealIDs []string
    AccountIDs []string
    EventIDs []string
    EdgeIDs []string
    OccurrenceIDs []string
}
type EvidenceResult struct { Meta Meta; Evidence Evidence }

type ActionTemplate struct {
    ActionID string
    ActionType string
    Definition string
    TaxonomyVersion string
}
type PolicyFacts struct {
    RequestedDiscountBPS Fact[int32]
    RequestOccurrenceID *string
    DecisionID *string
    AppliedContractID *string
    ApprovalStatus Fact[string]
    Approver ParticipantRef
    ApprovedDiscountBPS Fact[int32]
    ConsentStatus Fact[string]
    ConsentTargetNodeID *string
    ConsentScope Fact[string]
}
type ActionOccurrence struct {
    OccurrenceID string
    ActionID string
    EventID string
    Actor ParticipantRef
    Targets []ParticipantRef
    EventAt Date
    Status Fact[string]
    RawStatus *string
    OutcomeObserved Fact[string]
    OutcomeAt *Date
    AccountIDs []string
    DealIDs []string
    EvidenceIDs []string
    NodeIDs []string
    EdgeIDs []string
    Policy PolicyFacts
}
type ActionPrecedent struct {
    Relation string // current_deal / same_account_precedent / analog_precedent
    Occurrence ActionOccurrence
}
type RelevanceReason struct {
    Code string
    Detail string
    DealEvidenceIDs []string
    PrecedentEvidenceIDs []string
}
type ActionCandidate struct {
    Template ActionTemplate
    Precedents []ActionPrecedent
    RelevanceReasons []RelevanceReason
    Limitations []string
}
type CandidateOptions struct { RelevanceRuleVersion string; Limit int; Cursor *string }
type ActionCandidatesResult struct {
    Meta Meta
    Bounds BoundInfo
    RelevanceRuleVersion string
    ReturnedCount int
    TotalValidCount *int
    Items []ActionCandidate
}
type SelectedActionsResult struct {
    Meta Meta
    RelevanceRuleVersion string
    SelectedActionIDs []string
    Items []ActionCandidate
    Evidence []Evidence // lengkap untuk semua refs item, tidak silently truncate
}

// Interface kecil, masing-masing dipakai hanya oleh consumer yang perlu.
type DealFactsRepository interface {
    ListFacts(context.Context, SnapshotContext, DealListOptions) (DealFactsPage, error)
    FindFacts(context.Context, string, SnapshotContext) (DealFactsResult, error)
}
type GraphRepository interface {
    ReadGraph(context.Context, string, SnapshotContext, GraphOptions) (GraphResult, error)
}
type TimelineRepository interface {
    ReadTimeline(context.Context, string, SnapshotContext, TimelineOptions) (TimelineResult, error)
}
type EvidenceRepository interface {
    ReadEvidence(context.Context, string, SnapshotContext) (EvidenceResult, error)
}
type ActionRepository interface {
    ReadActionCandidates(context.Context, string, SnapshotContext, CandidateOptions) (ActionCandidatesResult, error)
    ReadSelectedActions(context.Context, string, SnapshotContext, []string, string) (SelectedActionsResult, error)
}

type ErrorCode string
const (
    NotFound ErrorCode = "not_found"
    NotVisible ErrorCode = "not_visible_at_snapshot"
    DataNotReady ErrorCode = "data_not_ready"
    AmbiguousReference ErrorCode = "ambiguous_reference"
    QueryFailed ErrorCode = "query_failed"
    InvalidSnapshot ErrorCode = "invalid_snapshot"
    InvalidQuery ErrorCode = "invalid_query"
    AccessDenied ErrorCode = "access_denied"
    ContextMismatch ErrorCode = "context_mismatch"
    InvalidSelection ErrorCode = "invalid_selection"
    BundleLimitExceeded ErrorCode = "bundle_limit_exceeded"
)
type LookupIssue struct { ID string; Code ErrorCode }
type RepositoryError struct {
    Code ErrorCode
    EntityKind string
    EntityID string
    Issues []LookupIssue
    Cause error // internal; tidak diserialisasi ke frontend
}
func (e *RepositoryError) Error() string { return string(e.Code) }
func (e *RepositoryError) Unwrap() error { return e.Cause }
```

Semua string parameter setelah `context.Context` yang berdiri sendiri adalah **dealID**, kecuali `ReadEvidence` menerima **evidenceID**. `ReadSelectedActions` menerima `(ctx, dealID, snapshot, selectedTemplateIDs, relevanceRuleVersion)`. Signature tanpa nama hanya menghemat pengulangan pada draft; penamaan parameter nyata dibekukan bersama. Method tidak mengekspos query text atau tipe driver.

Semantik operasi:

| Operasi | Jaminan O1 / perilaku consumer |
|---|---|
| `ListFacts` / `FindFacts` | List hanya deal visible dan authorized. Detail ID tidak ada berbeda dari ada tetapi belum dibuat pada cutoff. Semua facts membawa availability/provenance; list default order `deal_id` ascending. |
| `ReadGraph` | Root deal visible; focus event harus terkait context deal/account yang sah. Bounds dan node/edge consistency diperiksa. Unknown identity tidak dipaksa resolved. |
| `ReadTimeline` | Filter intersect snapshot/access; linked account-only context dilabeli. Event ID filter membantu UI focus. Stable date/ID order. |
| `ReadEvidence` | Satu unit evidence, field/span, claim date dan scope, tidak seluruh row/thread. Source yang tidak visible ditolak tanpa excerpt. |
| `ReadActionCandidates` | Template hanya dengan ≥1 precedent valid dan ≥1 relevance reason bersumber; exact count jika tidak truncated, otherwise jangan klaim total lengkap. |
| `ReadSelectedActions` | IDs adalah template IDs yang returned sebagai `action_id`. Unik; existence + relevance + cutoff + access divalidasi lagi di database, termasuk ID yang tidak berada di page pertama. Satu ID boleh untuk inspeksi detail; aturan compare 2–4 di O2. Batch invalid gagal secara atomik dengan kategori per ID, tidak silently menghapus/mengganti opsi. Semua evidence refs item valid ada dalam bundle. Bundle terlalu besar menghasilkan error eksplisit, bukan klaim complete. |

Tidak diperlukan method JEV/Score/Choice/Copilot pada repository data. O2 dapat menyusun model bundle dari deal facts + graph/timeline + selected/evidence hasil context yang sama. Readiness adapter dapat diperiksa saat wiring; interface status dataset terpisah hanya ditambah jika consumer membutuhkan, tanpa memperlebar keenam kebutuhan di atas.

JSON di C adalah **projection domain usulan**, bukan DTO HTTP yang sudah disetujui. Hanya `Fact` diberi tag contoh agar nullability jelas; O2 menentukan tag/output envelope final dan tidak menyerialisasi semua struct Go default apa adanya.

## C. Contoh JSON — ilustrasi kontrak, bukan hasil query dataset asli

Semua ID berawalan `TEST`, kutipan, nilai, digest placeholder dan tanggal kejadian di bawah adalah **DATA UJI / ilustrasi kontrak**. Nama file/kolom menggambarkan bentuk sumber aktual; record yang ditampilkan tidak diklaim ada di file kompetisi. Potongan bertanda `projection` hanya menampilkan field relevan, bukan response lengkap. Semua null tetap explicit.

### C1. Normal: fakta snapshot dengan provenance

```json
{
  "example_kind": "contract_illustration_not_dataset_query",
  "projection": "deal_fact_and_evidence",
  "meta": {
    "contract_version": "context-contract-v0.1-draft",
    "dataset_version": "TEST-MANIFEST-V1",
    "as_of": "2026-10-01",
    "max_as_of": "2026-10-01",
    "calendar_zone": "Asia/Jakarta",
    "context_id": "TEST-CTX-SNAPSHOT",
    "data_state": "ready",
    "limitations": [],
    "unknowns": []
  },
  "deal": {
    "deal_id": "TEST-DL-A",
    "account_id": "TEST-ACCOUNT-A",
    "potential_acv_idr": {
      "value": 1000000,
      "state": "known",
      "temporal_basis": "snapshot",
      "evidence_ids": ["TEST-EV-ACV"],
      "limitations": ["historical_value_unavailable"]
    }
  },
  "evidence": {
    "evidence_id": "TEST-EV-ACV",
    "source_file": "crm_deals.csv",
    "source_record_id": "TEST-DL-A",
    "record_id_kind": "native",
    "source_key": {"deal_id": "TEST-DL-A"},
    "source_checksum": "TEST-FILE-CHECKSUM",
    "source_field": "nilai_tahunan",
    "span": null,
    "source_date": null,
    "source_timestamp": null,
    "temporal_basis": "snapshot",
    "content_excerpt": "1000000",
    "scope_kind": "deal",
    "verification_state": "verified",
    "deal_ids": ["TEST-DL-A"],
    "account_ids": ["TEST-ACCOUNT-A"],
    "event_ids": [],
    "edge_ids": [],
    "occurrence_ids": []
  }
}
```

### C2. Unknown/ambiguous: tidak mengubah missing menjadi zero/consent

```json
{
  "example_kind": "contract_illustration_not_dataset_query",
  "projection": "unknown_and_identity",
  "planned_outlets": {
    "value": null, "state": "unknown", "temporal_basis": "snapshot",
    "evidence_ids": [], "limitations": ["source_field_missing"]
  },
  "participant": {
    "node_id": null,
    "raw_ref": "TEST-historical-address@example.invalid",
    "verification_state": "ambiguous",
    "match_method": "multiple_supported_candidates",
    "candidate_node_ids": ["TEST-PERSON-A", "TEST-PERSON-B"],
    "evidence_ids": ["TEST-EV-IDENTITY"]
  },
  "consent_status": {
    "value": null, "state": "unknown", "temporal_basis": "undated",
    "evidence_ids": [], "limitations": ["consent_evidence_not_found"]
  }
}
```

Candidate identity refs contoh ini harus dapat di-resolve dalam fixture yang sama; dua candidates tidak membuktikan dua orang itu benar-benar participant. Missing value tidak berubah menjadi `false`, `rejected`, atau `resolved`.

### C3. Historical: request visible, approval/application/outcome berikutnya disaring

```json
{
  "example_kind": "contract_illustration_not_dataset_query",
  "projection": "historical_action_and_stage",
  "as_of": "2026-09-10",
  "context_id": "TEST-CTX-HISTORICAL",
  "stage": {
    "value": null, "state": "snapshot_only", "temporal_basis": "snapshot",
    "evidence_ids": [], "limitations": ["stage_history_unavailable"]
  },
  "occurrence": {
    "occurrence_id": "TEST-OCC-REQUEST",
    "event_id": "TEST-EVENT-REQUEST",
    "event_at": "2026-09-09",
    "status": {
      "value": "requested", "state": "known", "temporal_basis": "event",
      "evidence_ids": ["TEST-EV-REQUEST"], "limitations": []
    },
    "outcome_observed": {
      "value": null, "state": "unknown", "temporal_basis": "event",
      "evidence_ids": [], "limitations": ["no_visible_outcome_evidence"]
    },
    "outcome_at": null,
    "decision_id": null,
    "applied_contract_id": null
  },
  "timeline_event_ids": ["TEST-EVENT-REQUEST"],
  "visible_evidence_ids": ["TEST-EV-REQUEST"],
  "recorded_at": null
}
```

Dalam fixture pengujian, approval setelah tanggal contoh ini dapat tersimpan di database tetap; projection historical tidak mengembalikan ID, kutipan, status, atau tanggalnya. Kalau occurrence sudah terlihat tetapi outcome belum ada pada cutoff, unknown bukan outcome gagal. Pada interface Go, link policy contoh ini berada dalam `occurrence.policy`; projection sengaja hanya memperlihatkan field terkait.

### C4. Kandidat hanya satu dan preseden analog

```json
{
  "example_kind": "contract_illustration_not_dataset_query",
  "projection": "candidate_count_and_precedent",
  "as_of": "2026-10-01",
  "returned_count": 1,
  "total_valid_count": 1,
  "truncated": false,
  "items": [{
    "template": {"action_id": "TEST-ACTION-A", "action_type": "FOLLOW_UP", "definition": "DATA UJI: tindak lanjut terdokumentasi", "taxonomy_version": "TEST-TAXONOMY-V1"},
    "precedents": [{
      "relation": "analog_precedent",
      "occurrence": {"occurrence_id": "TEST-OCC-ANALOG", "event_at": "2026-09-01", "account_ids": ["TEST-ACCOUNT-B"], "evidence_ids": ["TEST-EV-ANALOG"]}
    }],
    "relevance_reasons": [{"code": "same_documented_request_type", "detail": "DATA UJI: match bersumber", "deal_evidence_ids": ["TEST-EV-DEAL-REQUEST"], "precedent_evidence_ids": ["TEST-EV-ANALOG"]}],
    "limitations": ["precedent_is_not_current_deal_approval"]
  }]
}
```

O2 menampilkan tidak cukup opsi untuk comparison; tidak menambah opsi sintetis atau menjalankan Choice atas set satu. Akses ke `TEST-ACCOUNT-B` wajib diberikan server. Relevance reason bukan skor suitability atau bukti bahwa tindakan berhasil.

## D. Kontrak error dan hasil kosong

Error repository dapat diperiksa O2 dengan `errors.As(err, &repoErr)`; context cancellation/deadline tetap `errors.Is` melalui cause/unwrap. Pesan HTTP, status HTTP, redaksi, dan logging aman ditentukan O2, bukan draft O1. Jangan mengirim driver error, query, password, atau detail sumber di luar akses melalui `Cause`.

| Kategori usulan | Trigger | Kontrak O1 | Tanggung jawab O2 |
|---|---|---|---|
| `not_found` | ID tidak ada di dataset tetap | Bedakan dari snapshot invisibility; tidak return zero entity + nil error | HTTP mapping dan concealment existence untuk caller tidak berizin. |
| `not_visible_at_snapshot` | ID ada tetapi claim/deal belum tersedia pada cutoff | Tidak mengembalikan detail future; entity internal hanya untuk caller authorized | Respons aman tanpa menjelaskan outcome future. |
| `data_not_ready` | DB/schema/dataset version belum siap, core source integrity gagal | Bukan empty list; tidak menginisialisasi/import database dari request | Readiness/loading, konfigurasi dan operasi aplikasi. |
| `ambiguous_reference` | Lookup membutuhkan satu identity/link tetapi beberapa kandidat sah | Tidak memilih candidate pertama | Pilih respons koreksi/unknown sesuai use case. |
| `query_failed` | DB/query gagal setelah konfigurasi sah | Cause internal; bukan `not_found`, empty, atau insufficient evidence | HTTP mapping, retry/log sesuai error dan context. |
| `invalid_snapshot/invalid_query` | Tanggal >cutoff/format salah, bounds/filter tidak sah | Tolak defensif meski O2 memvalidasi input | Request validation dan DTO error. |
| `access_denied` | Target/scoped evidence tidak diotorisasi | Tidak mengembalikan excerpt/candidate refs terlarang | Otorisasi serta penyamaran not-found bila diperlukan. |
| `context_mismatch` | Versi/cursor/context tidak cocok | Tidak campur snapshot/versi atau reuse cursor | Ulang query dengan context server yang benar. |
| `invalid_selection` | ID selected duplicate, bukan kandidat relevan, atau lookup sebagian invalid | Batch fail atomik; `Issues` berisi kind per ID yang boleh diungkap | 2–4 unique untuk compare, redaksi issues, tanpa silent truncation. |
| `bundle_limit_exceeded` | Selected evidence tidak dapat dipenuhi lengkap dalam budget | Jangan mengirim bundle parsial sebagai lengkap | Batasi/revisi request dengan pesan eksplisit. |

Tidak semua ambiguity mematikan query: participant ambiguous pada graph/timeline muncul sebagai `ParticipantRef` + `unknowns`, dengan `err == nil` bila query lain tetap benar. Error ambiguous dipakai saat operasi memerlukan satu referensi pasti. List/timeline/candidates kosong yang memang lengkap → empty arrays, bounds lengkap, nil error; jangan mengartikan empty sebagai “tidak pernah terjadi”.

Jika selected template ada tetapi semua occurrence-nya future → `not_visible_at_snapshot`; jika visible tetapi tidak relevan → `invalid_selection`. Template yang tidak ada → `not_found`. Query batch dapat membawa kategori masing-masing melalui `Issues` tanpa mengembalikan sebagian item sukses sebagai comparison lengkap.

## E. Acceptance test handoff — spesifikasi, belum dijalankan

O1 menyiapkan contract fixtures terisolasi berlabel **DATA UJI**, lalu menjalankan suite yang sama terhadap adapter database setelah disetujui. O2 menguji HTTP/service/DTO memakai interface yang sama. Expected facts integration ditelusuri ke source IDs file kompetisi; hasil model tidak menjadi gold truth. Tidak ada pipeline ETL/reimport test otomatis yang harus dibangun untuk menyelesaikan kontrak ini.

| ID / calon test | Setup dan assertion yang wajib | Bukti handoff |
|---|---|---|
| T01 `DealAndAccountIDsAreDistinct` | Pada sumber kompetisi periksa `DL-001/P01` … `DL-005/P05`; lookup `P01` sebagai deal tidak otomatis diperlakukan `DL-001`. Semua FK account/owner references resolve atau readiness error. | Mapping expected native IDs dan laporan FK. |
| T02 `ReferencesResolveInSameContext` | Tiap edge source/target ada dalam graph hasil. Event→node/edge/evidence dan occurrence→event/evidence resolve pada context/scope sama; bounds tidak menghasilkan dangling edge. | Daftar path assertion untuk satu deal nyata + fixture truncation. |
| T03 `GraphTimelineEvidenceFocusAgree` | Event dari timeline dipakai sebagai `focus_event_id`; hasil graph memuat refs pendukung yang diperlukan dalam budget. Graph `event_ids` dapat dibaca lewat timeline event-ID filter, excerpt membuktikan claim. Invalid focus event lintas deal ditolak. | Shared IDs dan excerpt/field asal, bukan screenshot saja. |
| T04 `NoFutureLeakageAcrossAllFields` | Fixture event sebelum/saat/setelah cutoff; reply, approval, outcome, validity end dan consent setelah cutoff. Semua future IDs/text/status/links/summary disaring; source tanpa jam tidak diberi jam. Event pada tanggal cutoff termasuk. | Assertion atas graph, timeline, evidence, candidates, selected bundle. |
| T05 `SnapshotStageAndValuesStayUnknownHistorically` | Snapshot stage/owner/status/ACV/outlet tanpa history; sebelum snapshot `value:null` + limitation, current stage tidak dipakai untuk match/rank historical. Native stage_since bukan fabricated stage timeline. | Expected Fact states dan evidence access historical ditolak. |
| T06 `NullDoesNotBecomeZeroOrFalse` | Fixture missing/known-zero/ambiguous; ACV null vs 0, outlet null vs 0, consent null vs denied. Domain dan JSON O2 mempertahankan null serta state; unknown outcome tidak menjadi failed. | O1 fixture expected + O2 serialization assertions. |
| T07 `RequestApprovalApplicationRemainSeparate` | Fixture DATA UJI 10% requested→10% approved→14% requested, tanpa approval 14%. Request/decision/application memiliki IDs dan evidence terpisah; approval lama tidak diwariskan. >10% input warning O2; billing hanya applied jika link claim/value/scope terbukti. | Expected three-event sequence; jangan campur ke P02 kompetisi. |
| T08 `ActionCandidateHasVisibleSupportedPrecedent` | Setiap template candidate punya occurrence evidence valid dan relevance evidence deal+precedent; future-only catalog tidak muncul. 0/1 candidate returned persis; halaman truncated tidak mengaku total valid 0/1. | Native source paths ketika data tersedia; synthetic negative fixtures. |
| T09 `AnalogPrecedentDoesNotGrantApprovalOrConsent` | Occurrence akun B untuk deal akun A diberi analog label; approval/consent B tidak diwariskan ke A. Tanpa permission B tidak ada IDs/quotes B dalam candidates/evidence. | Scope allow/deny fixtures + policy linkage assertions. |
| T10 `AsOfChangeKeepsContextConsistent` | Ambil semua panel tanggal A lalu B dan kembali A. Meta context/date/version selalu sama dalam satu set; data A reproducible, future B tidak terbawa ke A. Cursor A ditolak pada B/versi/scope berbeda. | Context IDs, expected filtered IDs; UI atomic switch milik O2/UI owner. |
| T11 `AmbiguousIdentityIsNotVerified` | Email historis mismatch/multiple candidates dan employment boundary; node chosen null, state ambiguous, match methods/evidence jelas. Field `selesai` boundary diuji sesuai keputusan F, bukan asumsi diam-diam. | History/identity fixtures dan mapping source keys. |
| T12 `AccountOnlyEvidenceStaysAccountOnly` | Account dengan dua deals dan interaction tanpa deal_id. Tidak auto-attach ke keduanya/salah satu; result scope account dan uncertainty tetap. Single-deal fallback hanya jika aturan O2 disetujui serta direkam inferred. | Linkage rules + expected absent/present edges. |
| T13 `SelectedIDsRevalidatedAtomically` | Duplicate/unknown/invisible/irrelevant/unauthorized template IDs, termasuk ID sah di page selanjutnya. Seluruh invalid batch error dengan issues aman; selected evidence lengkap, tidak pad/substitute opsi. | Error expectations dan source refs bundle sukses. |
| T14 `ReadinessFailureIsNotEmptyData` | Dataset/schema belum siap, DB gagal, ID absent, ID future, ambiguity dan timeout. Kategori berbeda; deadline/cancel tidak berubah menjadi sukses kosong. | Adapter negative-path tests + O2 HTTP tests terpisah. |
| T15 `BusinessDatabaseRemainsStatic` | Query sequence berulang/tanggal berbeda; IDs/data version tetap; tidak ada business writes/request-time CSV reads. DB business credential dibatasi read jika engine mendukung. Persiapan snapshot sama menghasilkan registry IDs konsisten lewat verifikasi artefak, tanpa pipeline baru. | Snapshot manifest/ID registry; read-only adapter integration check. |

Setelah implementasi disepakati, calon lokasi tests di `backend/repository/*_test.go` dan fixture terisolasi di `backend/repository/testdata/`. Perintah standar `go test ./...` dari `backend/`; live database tests memakai konfigurasi opt-in yang disepakati dan melaporkan skip sebagai skip. JEV live/HTTP/E2E bukan bukti lulus uji data O1. Target kinerja PRD tetap target engineering sampai benchmark benar-benar diukur.

## F. Keputusan yang benar-benar perlu dijawab Orang 2

Ini pertanyaan review kontrak, bukan pengulangan keputusan ETL. Tidak menanyakan apakah perlu pipeline: **database statis tanpa pipeline sudah ditetapkan user**.

| ID | Keputusan belum pasti | Rekomendasi O1 dan alasan | Yang terblokir |
|---|---|---|---|
| F01 | Tim sudah memilih engine/instance DB apa dan bagaimana O1 mendapat schema/data statis yang disiapkan? | Jika tim mengonfirmasi Aura tersedia, pakai satu Aura graph; kontrak tetap engine-independent. Serahkan akses/config secara aman, tanpa credential dalam dokumen. Diskusi Aura sebelumnya belum bukti koneksi/data siap. | Adapter konkret/config dan acceptance database nyata; bukan draft logical contract. |
| F02 | Bagaimana migrasi dari `DealRepository/models.Deal` existing ke facts nullable? | Tambah `DealFactsRepository` dahulu dan migrasi consumer eksplisit setelah review. Jangan mengisi historical unknown dengan zero. O2 memiliki perubahan `models/deal.go`, services dan HTTP DTO. | Typed facts source dan compatibility adapter existing. |
| F03 | Date-only, zona kalender, dan makna employment `selesai`? | Date-only dengan seluruh hari `Asia/Jakarta`, cutoff maksimum 2026-10-01; internal interval half-open. Usulan selesai terisi adalah hari terakhir inklusif, raw date tetap disimpan. README belum menjelaskan boundary end terisi. | Boundary tests dan normalisasi tanggal legacy `time.Time` ke Date. |
| F04 | Link account-only interaction ke deal, unresolved identity, dan organisasi luar? | Jangan auto-attach; pakai explicit evidence/rule reviewed. Jika single-deal fallback dibolehkan, label inferred dan simpan alasan/version. Organisasi luar tetap locator bersumber; email alias historis tidak ditebak dari nama. | Rules identity/linkage yang dapat dikonsumsi tanpa dua mapping berbeda. |
| F05 | Scope akses demo, company policy evidence, dan izin analog antar-akun? | O2 memasok allowlist server; current deal permission terpisah dari izin analog. Company evidence perlu klasifikasi/izin eksplisit. Untuk denominator ACV snapshot penuh, O2 memerlukan akses input sesuai PRD yang tidak dibatasi filter UI; jangan membuka semua sumber hanya demi denominator. | Evidence/analogue exposure, input scoring denominator. |
| F06 | Taksonomi template, relevance rule minimum, dan consent/approval fields mana yang dipakai O2? | Minimum ≥1 occurrence dan ≥1 reason bersumber; versions tetap saat database disiapkan. Status raw dipertahankan; PolicyFacts per request/target, bukan template. Eligibility compare/warning milik O2; unknown izin boleh terlihat sebagai unknown sesuai kebijakan compare yang disepakati. | Candidate acceptance + selected evidence handoff, tanpa membuat suitability O1. |
| F07 | Query budgets, focus, paging dan selected evidence cap? | Usulan graph default depth 1, max depth 2, ≤150 nodes (target PRD), max 300 edges; timeline/list/candidates page 50. Selected bundle lengkap atau error; angka cap evidence disepakati dari kebutuhan O2 dan ukuran data, jangan ditentukan secara rekaan. | Default bounds/DTO pagination dan bundle-limit behavior. |
| F08 | Setuju versi/meta/state/error categories dan HTTP mapping batch? | Pakai per-field Fact, shared context_id, explicit truncation, typed repository errors; batch selected atomic. O2 membekukan DTO/HTTP statuses dan redaksi unauthorized detail. | Consumer fixtures/error contracts, tidak memblokir schema diskusi. |

Pertanyaan owner UI, provider JEV/Copilot, ranking, dan cache implementation tidak dibekukan O1 di kontrak data ini. O2 tetap memakai fitur PRD dan koordinasi owner yang diperlukan; draft ini tidak mengubah scope produk.

## 2. Bagian yang siap disepakati

Prinsip berikut memiliki dasar langsung pada arahan user/PRD, sehingga siap menjadi titik persetujuan tanpa memilih vendor:

- Database bisnis statis disiapkan sekali, tanpa pipeline/refresh; semua interface read-only.
- Deal/account berbeda; IDs stabil, typed graph dan source provenance; tidak ada fake recorded_at/history/source IDs.
- Cutoff maksimum 2026-10-01; snapshot/access yang sama untuk graph, timeline, evidence, dan action bundles.
- Unknown/null/ambiguous tetap eksplisit; request, proposal, approval, rejection, application berbeda; preseden tidak memberi izin.
- Template hanya dengan occurrence berbukti; candidate <2 tetap jumlah sebenarnya; JEV/scoring/comparison/Copilot/cache tetap O2.
- Signature existing tidak berubah diam-diam; source/migration/arsitektur bersama belum diubah.

**Siap disepakati tidak berarti sudah disetujui Orang 2.** Detail teknis di A–B tetap draft.

## 3. Bagian yang masih membutuhkan jawaban Orang 2

Prioritas review: F02 compatibility dan nullable model; F03 boundary waktu; F04 identity/deal linkage; F05 akses; F08 shape/error contract. Berikutnya F06 candidate rules dan F07 budgets. F01 diperlukan sebelum adapter konkret dan pengujian database; pembacaan database siap harus dapat dibuktikan, bukan diasumsikan.

Catat persetujuan per F-ID, rekomendasi diterima/diubah, versi keputusan, dan reviewer O2. Sampai review tersebut, label dokumen tetap **DRAFT — menunggu persetujuan Orang 2**. Tidak membuat source, migration, atau mengubah arsitektur bersama berdasarkan draft ini.

## 4. Handoff minimum agar Orang 2 dapat mulai membangun API

1. **Sekarang:** review A–B, examples C dan errors D sebagai bahan membekukan null/snapshot/scope interface. O2 dapat menyiapkan routes/DTO/service terhadap test-only stubs tanpa menganggap adapter/data sudah siap.
2. **Sesudah persetujuan kontrak:** O1 menyerahkan tipe domain/interface sesuai package langsung `models/` dan `repository/`, contract fixtures berlabel DATA UJI, daftar expected errors, dan suite acceptance E. O2 melakukan wiring; legacy changes sesuai F02.
3. **Untuk satu vertical slice nyata:** database statis siap + manifest versi/checksum/ID registry, config adapter tanpa secret, satu deal native dengan graph/timeline/evidence dan candidates bila benar-benar ada; source field/locator bisa diperiksa. Candidate 0/1 tetap valid sebagai keterbatasan slice.
4. **Bukti integrasi:** perintah menjalankan adapter tests dan report pass/fail/skip aktual; query date A/B dengan context sama lintas panel, unknown/ambiguous dan failure paths. Source fixture bukan hasil database; scaffold health bukan bukti database ready.
5. **Untuk mulai JEV/Copilot:** O2 mendapat facts/evidence/selected bundles yang konsisten dan snapshot-safe serta scope valid; O2 membangun provider/cache/scoring sendiri di atas kontrak ini. Tidak ada source mapping ulang atau endpoint penarikan dataset baru.

Status penyerahan dokumen ini: draft review tersedia; source/backend, migration, database dan dokumen arsitektur/development plan bersama belum diubah. Acceptance tests di E adalah rencana handoff, belum bukti adapter lulus.
