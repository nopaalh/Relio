# Relio — Audit sumber dataset tetap

**Tanggal pemeriksaan:** 9 Oktober 2026.  
**Status:** observasi source files telah diperiksa; bukan persetujuan kontrak, schema database, data publish, atau acceptance adapter.  
**Arahan terbaru user:** database bisnis statis, disiapkan sekali; tanpa pipeline ETL, connector, scheduler, refresh otomatis, atau perubahan file dataset.

Laporan ini mengerjakan bagian O1 yang dapat berjalan sambil Orang 2 belum tersedia. Acuan: [READMEHack](../../../READMEHack.md), [PRD Final v1.0](../../../Relio_PRD_Final_MVP_v1.0_Agent_Ready_ID.md), [development plan](development-plan.md), dan [draft kontrak](data-repository-contract-draft.md). Rencana lama yang mewajibkan pipeline ETL disesuaikan oleh arahan user tersebut; tidak dibangun dalam pekerjaan ini.

Data sumber ada di `../../../Datasets/`. Dataset kompetisi dinyatakan sintetis oleh README; angka di laporan ini berasal dari file kompetisi tersebut, bukan fixture tambahan. Machine-readable observations, checksum SHA256 seluruh file, missing-field counts, rentang tanggal, dan source slice ada di [source-audit-snapshot.json](source-audit-snapshot.json). Mapping field dan bukti P04 ada di [source-mapping.md](source-mapping.md).

## 1. Ringkasan hasil terukur

| Pemeriksaan | Hasil |
|---|---:|
| File CSV/JSONL diparse seluruhnya | 15 |
| Total logical source records, tidak termasuk CSV header | 229.627 |
| Duplicate native/candidate composite keys yang diperiksa | 0 |
| Pemeriksaan FK nonkosong, termasuk referensi versi aplikasi | 685.109 |
| FK nonkosong dengan target tidak ditemukan | 0 |
| Required references kosong pada field yang diperiksa | 0 |
| Malformed rows pada parser / format field yang diperiksa | 0 |
| Invalid ISO date/month pada field bertanggal yang diperiksa | 0 |
| Invalid/negative integer pada field numerik yang diperiksa | 0 / 0 |
| Pointer champion berbeda dari current account contact | 1 |
| Pasangan interval employment strictly overlapping | 10 |
| Current contact tanpa tepat satu history row sesuai account/jabatan pada snapshot | 0 |
| Pemakaian versi aplikasi sebelum tanggal rilis native pada usage/tickets | 0 |

Angka nol berarti tidak ditemukan oleh pemeriksaan yang disebutkan; bukan bukti seluruh semantik/business linkage sudah benar. Composite history key yang diperiksa memakai semua enam kolom, bukan keputusan stable ID final. Overlap memakai `next.mulai < previous.selesai`, atau end terbuka; makna end-day tetap belum diputuskan. Overlap dapat berarti pekerjaan paralel dan tidak otomatis invalid.

## 2. Inventaris actual vs README

| Source file | Logical rows actual | Acuan README | Peran |
|---|---:|---:|---|
| `crm_accounts.csv` | 45 | 45 | Account IDs, pelanggan/prospek, snapshot facts |
| `crm_contacts.csv` | 160 | 160 | Current contact IDs/email/account/jabatan |
| `contact_employment_history.csv` | 217 | ±200 | Riwayat organisasi/jabatan dan interval |
| `crm_deals.csv` | 22 | ±25 | Deal IDs, account FK, snapshot komersial |
| `employees.csv` | 10 | 10 | Internal actor/owner/approver references |
| `interactions.jsonl` | 350 | 350 | Dated email/internal email/meeting evidence |
| `outlets.csv` | 620 | 620 | Outlet/account mapping |
| `product_usage_daily.csv` | 226.300 | ±226 ribu | Observed daily usage pelanggan existing |
| `feature_usage_monthly.csv` | 1.178 | ±1.100 | Observed feature/account/month records |
| `contracts_billing.csv` | 40 | 40 | Snapshot kontrak, diskon, renewal |
| `decision_log.csv` | 30 | 30 | Dated keputusan dan field status terkait |
| `support_tickets.csv` | 640 | 640 | Enrichment opsional P0 |
| `bugs.csv` | 4 | 4 | Enrichment opsional P0 |
| `releases.csv` | 3 | 3 | Native release date/version lookup |
| `features.csv` | 8 | 8 | Roadmap snapshot dan quarter/free-text target |

README menggunakan perkiraan untuk beberapa file; selisih dari perkiraan bukan missing-data error. Account terdiri dari **40 pelanggan + 5 prospek**. Deal status native: **16 Menang, 1 Kalah, 5 Terbuka**. Interaksi native: **275 email, 71 catatan_meeting, 4 email_internal**; satu interaction memiliki account kosong.

## 3. Integritas referensi dan batas kesimpulan

Pemeriksaan membership FK meliputi account owners/champions, current contact account, history contact/account, deal account/owner, outlet account, usage account/outlet/version, monthly feature ID, contract account/decision, decision actor/deal/evidence/feature, interaction account/reply, ticket account/outlet/contact/bug/version, dan bug version/feature. Empty optional fields dihitung sebagai missing/absence, bukan invalid FK atau angka nol.

Checks cross-field juga memeriksa:

- Account outlet pada usage/ticket sesuai `outlets.account_id`; tidak ditemukan mismatch.
- Semua explicit meeting participant ID ditemukan di contacts/employees. History row yang sesuai contact dan account pada tanggal meeting ditemukan menurut strict-end diagnostic; ini bukan pembekuan identity rule.
- 111 native reply links resolve; tidak ditemukan parent bertanggal setelah reply atau parent account berbeda. Subject `Re:` tanpa `membalas_id` tidak menjadi reply link otomatis.
- Decision native deal/account consistent; satu native interaction evidence link account/date consistent. Ini memastikan reference, bukan semua semantics request→approval sudah sah.
- Satu contract native decision link memiliki account dan nilai diskon yang cocok. Tidak ditemukan kontrak >10% tanpa `decision_id`; hasil ini tidak membuktikan semua approval lain atau historis role VP.
- Nilai tahunan 40 kontrak cocok dengan field harga per outlet/bulan × outlet kontrak ×12 × diskon native; tidak dipakai untuk mengisi ACV prospek atau mengklaim revenue realized.

Referensi yang resolve belum sama dengan authorization, provenance field/span lengkap, valid-at-event identity, atau approval scope yang berlaku pada request baru.

## 4. Temuan yang perlu dipertahankan dalam graph

### DQ01 — Champion C01 stale terhadap current employment

`crm_accounts[account_id=C01].champion_contact_id = K017`; `crm_contacts[contact_id=K017].account_id_saat_ini = P01`. History K017 mencatat C01 pada **2021-03-01 … 2026-08-15**, kemudian P01 sejak **2026-09-01**. Current job adalah GM Operations pada P01.

Jangan memperbaiki CSV atau menganggap K017 masih champion aktif C01 hanya dari pointer tersebut. Simpan pointer sebagai snapshot CRM claim dengan limitation, dan gunakan employment bertanggal secara terpisah. User/Orang 2 belum menyetujui rule label/status edge champion.

### DQ02 — Satu email historis unmatched, muncul enam kali

Ada **629 nonblank address occurrences** pada field `dari/ke`; **623** memiliki satu exact match di email contacts/employees saat ini, **6** tidak memiliki current match, dan tidak ditemukan duplicate current-email candidates dalam pemeriksaan ini.

Keenam unmatched occurrences memakai satu alamat `rina.hapsari@kopilintas.co.id`: **I0051, I0066, I0159, I0223, I0224, I0290**. Current email K017 adalah `rina.hapsari@mandalaritel.co.id`. History C01 dan pesan pamit I0290 membantu review identitas, tetapi history file tidak menyimpan email alias. Jangan mengisi alias verified otomatis dengan similarity nama/local-part. Exact current-email match lain juga tidak otomatis membuktikan email/role itu berlaku sepanjang history.

### DQ03 — 10 strict overlap pairs pada history

Contact IDs: **K035, K039, K041, K084, K087, K091, K092, K113, K117, K148**. Masing-masing memiliki satu pasangan interval overlap dalam data yang diperiksa. Full source row dan physical line locators ada di JSON audit.

Ini 10 pasangan waktu, bukan 10 duplicate persons atau bukti data pasti salah. Bisa menjadi jabatan paralel atau kualitas history yang perlu interpretasi. Tidak truncate tanggal, merge person, atau memilih satu organisasi secara diam-diam. Ada 56 contact dengan lebih dari satu history row; overlap hanya bagian dari kelompok tersebut.

### DQ04 — Link keputusan/preseden sebagian besar account-only

Decision memiliki native `deal_id` hanya **3/30** record: `D-2025-02→DL-006`, `D-2025-06→DL-007`, `D-2025-11→DL-008`. Native `bukti_interaction_id` hanya **1/30**: `D-2025-11→I0061`. Kontrak dengan native `decision_id` hanya **1/40**: `K-C01→D-2025-11`.

Account C23 memiliki dua deals DL-006 dan DL-007; account-only interactions C23 tidak otomatis ditautkan ke keduanya. `K-C23.mulai` yang sama dengan tanggal stage Won DL-007 tidak menjadi native deal/decision FK. Matching tanggal/nilai bisa menjadi bahan review, bukan join sah tanpa aturan/bukti yang disepakati.

Kelima deals prospek tidak memiliki native `decision_log.deal_id` yang menunjuknya. Kalimat yang benar: **tidak ditemukan explicit decision link ke kelima deal dalam file yang diperiksa**. Jangan menyimpulkan tidak pernah ada approval/komunikasi di dunia bisnis.

### DQ05 — Record lama mengandung retrospective fields

Filter `decision_log.tanggal <= as_of` saja berpotensi membocorkan hasil masa depan:

| Sumber | Tanggal record | Field yang perlu pemisahan temporal |
|---|---|---|
| `D-2025-02` | 2025-03-04 | `alasan` menyebut deal kemudian kalah; CRM snapshot DL-006 stage_since 2025-03-14, tanpa stage log lengkap. |
| `D-2025-06` | 2025-08-12 | `alasan` menyebut deal menang Sep 2025; CRM snapshot DL-007 stage_since 2025-09-08. |
| `D-2024-05` | 2024-08-15 | `status_janji = Ditepati (rilis Feb 2026)`; tidak berlaku sebagai completion pada Agustus 2024. |
| `D-2026-01` | 2026-01-20 | `status_janji = Ditepati (rilis Mei 2026)`; sumber hanya memberikan bulan, bukan hari completion yang tepat. |

Tidak mengarang tanggal completion dari awal/akhir bulan atau `stage_sejak`. Untuk historical bundle, field/quote yang bersifat retrospective perlu disembunyikan atau dibatasi sampai punya temporal support yang sah. Pemisahan field/span dan lifecycle adalah pekerjaan setelah kontrak disetujui; audit ini mengidentifikasi risikonya, belum membuat evidence IDs atau schema.

### DQ06 — Future planning berbeda dari future occurrence

Tidak ditemukan event dates sesudah cutoff 2026-10-01 pada interaction, keputusan, daily usage, ticket creation/resolution, bug creation/resolution, atau release fields yang diperiksa. Namun **40/40 tanggal renewal kontrak** berada sesudah cutoff, rentang **2026-11-05 … 2027-09-08**. Itu rencana/tanggal kontraktual dalam snapshot, bukan bukti renewal telah terjadi; belum membuktikan kapan rencana tersebut pertama diketahui.

Roadmap berisi quarter seperti `2026-Q3`, `2027-Q1`, string `Belum ditetapkan`, dan blanks. Ini bukan malformed ISO dates dan tidak dinormalisasi menjadi deadline hari fiktif. `FEAT-07.target_terkini = Belum ditetapkan`; jangan memaksa deadline dari target awal Q3.

## 5. Null/missing yang terukur

| Sumber/field | Blank records | Interpretasi yang harus dijaga |
|---|---:|---|
| Prospek `paket/champion_contact_id/nps_terakhir/health_score_dashboard` | 5 masing-masing | Absence snapshot fields, bukan zero NPS/health atau champion pasti tidak ada. |
| History `account_id` | 55 | Organisasi luar menurut README; nama organisasi tetap ada. |
| History `selesai` | 160 | Masih menjabat menurut README; end-day rule terisi tetap perlu review. |
| Decision `deal_id` / `bukti_interaction_id` | 27 / 29 | Jangan membuat linkage tanpa bukti. |
| Decision `nilai` | 2 | Tidak otomatis diskon 0%; keduanya janji_fitur. |
| Contract `decision_id` | 39 | Missing link, bukan rejected/no approval. |
| Interaction `account_id` | 1 | Internal/company evidence perlu scope explicit sebelum exposure. |
| Interaction `peserta` / `ke` | 279 / 71 | Email participants dari addresses, bukan fabricated meeting IDs; meeting `ke` boleh blank. |
| Daily usage `transaksi_offline_tersinkron` | 215.350 | Tepat 590 outlet offline-disabled ×365 hari; blank bukan zero offline usage. |
| Ticket `pelapor_contact_id` / `diselesaikan` | 614 / 36 | Reporter/resolution unknown jika tidak terdokumentasi. |

Semua 620 outlet memiliki 365 unique observed date records; penggunaan hanya pada 40 pelanggan existing, tidak ada rows akun P01–P05. Total 226.300 =620×365; ini bukti coverage tanggal record, bukan bukti tiap event POS dunia nyata tercatat lengkap. Semua 30 offline-enabled outlets memiliki nonblank offline values pada date records; tidak ditemukan offline value terisi pada outlet offline-disabled.

Monthly feature usage mempunyai 1.178 unique account/feature/month keys dalam 2025-10 … 2026-09. Absence kombinasi account/feature/month bukan pengguna aktif nol; audit tidak mengisi grid kosong atau mempublikasikan aggregate bisnis baru.

## 6. Lima prospek — facts snapshot dan source context

Nilai di bawah adalah **CRM snapshot**, bukan history nilai/stage. Jumlah interaksi dihitung lewat native `interactions.account_id`; belum menjadi daftar deal-specific yang disetujui.

| Account → Deal | Stage snapshot | Owner native | Rencana outlet | Potensi ACV Rp | Account interactions |
|---|---|---|---:|---:|---|
| P01 → DL-001 | Proposal | E06 | 60 | 252.000.000 | 4: I0279, I0310, I0325, I0343 |
| P02 → DL-002 | Demo | E07 | 15 | 63.000.000 | 4: I0269, I0296, I0322, I0348 |
| P03 → DL-003 | Discovery | E08 | 9 | 37.800.000 | 1: I0334 |
| P04 → DL-004 | Negosiasi | E06 | 35 | 147.000.000 | 3: I0284, I0314, I0335 |
| P05 → DL-005 | Lead | E07 | 40 | 168.000.000 | 0 |

`crm_deals.outlet` cocok dengan `crm_accounts.jumlah_outlet` pada kelima prospek. Tidak ada nilai missing pada field deal ID/account/owner/stage/created/outlet/annual value/status yang diperiksa; tetap desain nullable untuk historical/unknown sesuai draft. P05 empty account interactions bukan bukti Sales tidak pernah menghubungi dan tidak boleh berubah menjadi kesiapan level 0.

## 7. Metode dan verifikasi

Pemeriksaan memakai diagnostic Python stdlib sekali jalan di direktori temporary di luar repo. Parser membaca file secara read-only; daily usage di-stream. Tidak ada koneksi DB/network, transform/publish dataset, automatic ingestion, scheduler, source changes, atau import saat backend startup. Field schema/header, date/month parsing, integers, FK membership, duplicate keys, cross-field references, reply chronology, offline consistency dan source date filters diperiksa. Record-level semantics, identity alias resolution dan taxonomy tidak diotomatisasi.

Checksum output mematok **observational snapshot** ini, bukan final `dataset_version` yang sudah disetujui. `source_line` dalam diagnostic JSON adalah baris fisik untuk inspeksi file; bukan stable business ID, bukan provenance ID baru, dan tidak digunakan menjadi event/edge ID.

Verifikasi ulang ukuran input dan row counts bisa dijalankan dari folder HeketonPens tanpa menulis source:

```powershell
$sourceAudit = Get-Content -Raw -Encoding UTF8 -LiteralPath 'Relio\backend\docs\source-audit-snapshot.json' | ConvertFrom-Json
foreach ($sourceFile in $sourceAudit.files.PSObject.Properties) {
    $sourcePath = Join-Path 'Datasets' $sourceFile.Name
    $actualHash = (Get-FileHash -Algorithm SHA256 -LiteralPath $sourcePath).Hash.ToLowerInvariant()
    if ($actualHash -ne $sourceFile.Value.sha256) { throw "Source changed: $($sourceFile.Name)" }
    if ($sourceFile.Name.EndsWith('.csv')) {
        $actualRows = (Import-Csv -Encoding UTF8 -LiteralPath $sourcePath | Measure-Object).Count
    } else {
        $actualRows = (Get-Content -Encoding UTF8 -LiteralPath $sourcePath |
            Where-Object { $_.Trim() } | ForEach-Object { $_ | ConvertFrom-Json } | Measure-Object).Count
    }
    if ($actualRows -ne $sourceFile.Value.rows) { throw "Row count changed: $($sourceFile.Name)" }
}
```

Perintah di atas memverifikasi hashes/counts, bukan mengulang semua pemeriksaan semantic/cross-field. Daftar scope/methodology dan measured checks tersimpan di JSON agar hasil yang benar-benar diperiksa dapat dibedakan dari yang belum diperiksa.

## 8. Handoff yang siap dan pekerjaan yang tetap menunggu

**Siap sekarang:** input hashes/counts, verified native IDs/FK observations, missing-field profile, daftar ambiguity/history traps, lima prospect source contexts, dan slice P04 dengan hasil date filter atas record asli. Orang 2 dapat memakai ini untuk review error/unknown states serta bahan memilih real integration case; source-audit JSON bukan fixture HTTP siap deploy.

**Tetap menunggu kontrak inti:** nullable model/signature compatibility, date-end boundary, account-only→deal linkage, scope akses, evidence/ID schema, taxonomy/relevance dan query bounds. Engine belum dikunci. Tidak membuat adapter, migration, SQL/Cypher, database writes, canonical graph IDs, atau mengubah kontrak/arsitektur bersama berdasarkan audit ini.

**Belum diverifikasi:** database tersedia/siap, temporal query adapter, HTTP API untuk context, output JEV/Copilot, action eligibility lengkap, policy/consent acceptance, runtime/performance/E2E. Source audit berhasil dibaca tidak berarti MVP selesai.
