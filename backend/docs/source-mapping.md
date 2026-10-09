# Relio — Mapping sumber O1 dan slice nyata P04

**Tanggal:** 9 Oktober 2026.  
**Status:** observasi field/record dataset; rekomendasi consumer masih **DRAFT — menunggu persetujuan Orang 2**. Tidak membuat schema, adapter, graph/evidence IDs turunan, atau mengubah `DealRepository` existing.

Acuan: [READMEHack](../../../READMEHack.md), [PRD Final v1.0](../../../Relio_PRD_Final_MVP_v1.0_Agent_Ready_ID.md), [draft kontrak](data-repository-contract-draft.md). Angka dan source records dapat diperiksa di [audit report](dataset-audit.md) dan [JSON observations](source-audit-snapshot.json). Dataset FIX dan database bisnis statis disiapkan sekali; tidak ada pekerjaan ETL/connector/scheduler pada mapping ini.

## 1. Aturan membaca mapping

`Native` berarti field/ID memang ada di source, bukan claim bahwa nilai itu diketahui sepanjang history. `Snapshot` berarti tersedia pada file snapshot 2026-10-01; history nilai tidak diisi retroaktif. `Dated record` memakai tanggal yang sumber simpan; tiap field mutable/retrospective tetap perlu ditinjau temporalnya. `Candidate linkage` adalah bahan review, belum relasi graph disetujui.

Sumber tanpa native row ID tetap memakai **field tuple untuk inspeksi**, bukan source ID buatan. Nama model kontrak di tabel adalah kebutuhan semantic, bukan deklarasi Go yang sudah dibuat. O2 tidak perlu mem-parsing CSV kembali untuk request API ketika adapter O1 tersedia.

## 2. Mapping facts inti

| Kebutuhan consumer | Field sumber native | Provenance minimal | Temporal / batas mapping |
|---|---|---|---|
| Identitas deal/account | `crm_deals.deal_id,account_id` | Source row native `deal_id`; FK `crm_accounts.account_id` | Identitas tidak boleh disamakan; metadata akun tidak otomatis historis. |
| Deal type | `crm_deals.tipe` | Row + field | Snapshot unless chronology ada; tidak infer dari prefix. |
| Deal owner | `crm_deals.owner_id` | Native employee ID; `employees.employee_id` | Jangan substitusi account owner atau mengarang owner history. |
| Stage/age context | `crm_deals.stage,stage_sejak` | Masing-masing field row | Snapshot stage + anchor; bukan log lengkap. Historical stage belum otomatis known. |
| Created date | `crm_deals.dibuat` | Row + field | Dated creation claim; date-only, bukan midnight timestamp. |
| Planned outlet | `crm_deals.outlet` dan `crm_accounts.jumlah_outlet` | Kedua sumber jika cek konflik | Cocok pada lima prospek; snapshot planned, bukan actual usage. |
| Potential ACV Rp | `crm_deals.nilai_tahunan` | Row + field | Integer rupiah snapshot; bukan billing revenue/margin. |
| Current deal status | `crm_deals.status` | Row + field | Snapshot kecuali bukti lifecycle lain; tidak isi Lost/Won historical dari current row. |
| Current contact info | `crm_contacts.contact_id,nama,email,account_id_saat_ini,jabatan_saat_ini` | Native contact row dan field terpisah | Current values, bukan historical email alias registry. |
| Internal people | `employees.employee_id,nama,jabatan,email` | Native employee row | Actor ID/email lookup; historical role validity tidak tersedia sebagai log. |
| Employment | `contact_employment_history.contact_id,account_id,organisasi,jabatan,mulai,selesai` | Tidak ada native row ID; raw tuple + checksum audit untuk inspeksi | Date boundaries unapproved; external organization account kosong bukan broken FK. |
| Interaction/event context | `interactions.interaction_id,tanggal,tipe,account_id,subjek,isi` | Native interaction ID; kutipan/field | Native account scope. Event kategori tambahan/claim splitting belum difinalkan. |
| Participants/sender/recipient | `dari,ke,peserta` | Native email/meeting ID fields + source refs kontak/history | Meeting IDs exact; email identity tetap perlu event-time rule. |
| Reply chain | `membalas_id` | Native interaction FK | Resolve hanya record <= cutoff; subject `Re:` tidak cukup. |
| Decision facts | `decision_log.decision_id,tanggal,tipe,keputusan,nilai,alasan` | Native decision ID, field evidence | Status/quote tidak boleh mengambil retrospective outcome tanpa temporal support. |
| Decision actor/deal/evidence | `diminta_oleh,diputuskan_oleh,deal_id,bukti_interaction_id` | Native FK bila terisi; empty tetap empty | Banyak decision account-only; approval harus cocok request scope/nilai. |
| Contract application context | `contracts_billing.contract_id,account_id,mulai,diskon_pct,decision_id` | Contract row dan decision row bila native link ada | Link decision ≠ semua request approved; application scope/date perlu ditinjau. |
| Renewal plan | `contracts_billing.tanggal_renewal` | Field row | Future planned date boleh menjadi konteks snapshot; jangan membuat completed renewal event. |
| Usage analogue | `product_usage_daily.tanggal,outlet_id,account_id,jumlah_transaksi` | Native outlet FK; logical `(tanggal,outlet_id)` diagnostic key | Existing accounts only; transaksi count ≠ uang/profit; aggregate schema belum dibuat. |
| Offline value | `transaksi_offline_tersinkron`, `outlets.mode_offline_aktif` | Field + outlet mode | Blank disabled ≠ zero; jangan imputasi. |
| Monthly feature use | `bulan,account_id,feature_id,pengguna_aktif` | Composite key sumber | Month precision; combination absent ≠0; jangan memakai seluruh month untuk partial-month cutoff tanpa rule. |
| Optional support/product context | Tickets, bugs, releases, features | Native IDs/field references | P0 enrichment bila relevan; release date native, roadmap quarter/free text preserved. |

## 3. Native joins vs candidate joins

| Relation yang dibutuhkan | Yang benar-benar ada | Yang belum boleh diasumsikan |
|---|---|---|
| Deal→Account | `crm_deals.account_id` native FK | Current account attributes selalu known sebelum snapshot. |
| Deal→Employee | `crm_deals.owner_id` native FK | Ownership sepanjang history atau role VP sepanjang history. |
| Interaction→Account | `interactions.account_id` native FK bila nonblank | Interaction→Deal native FK; field `deal_id` tidak ada. |
| Decision→Deal | `decision_log.deal_id` bila terisi | Account-only decision diberi deal ID hanya karena akun punya satu/dua deal. |
| Decision→Interaction | `bukti_interaction_id` bila terisi | Subject/text similarity otomatis sama request. |
| Contract→Decision | `decision_id` bila terisi | Tanggal/nominal sama sebagai pengganti native link. |
| Meeting→Person | `peserta` contact/employee IDs | Quote nama lain di body otomatis verified participant ID. |
| Email→Person | Exact current-email lookup + history dapat memberi candidate support | Historical alias otomatis verified karena nama/local-part sama. |
| ActionTemplate→Occurrence | Sumber quote/request/decision dapat menjadi bahan katalog | Semua buyer request pasti merupakan tindakan Sales yang layak dibandingkan, atau outcome Won menjadi keberhasilan kausal. |

Canonical graph edge/event/evidence IDs belum dibuat. Native source ID dipakai di dokumen ini agar keputusan namespace/claim identity tidak dikunci sebelum review O2.

## 4. Slice P04 / DL-004 — source yang benar-benar tersedia

### 4.1 Deal dan account snapshot

| Source locator | Field | Nilai native | Batas |
|---|---|---|---|
| `crm_deals[deal_id=DL-004]` | `account_id` | `P04` | Native FK. |
| Row yang sama | `dibuat` | `2026-08-05` | Tanggal creation, jam tidak tersedia. |
| Row yang sama | `owner_id` | `E06` | Current owner snapshot. |
| Row yang sama | `stage` / `stage_sejak` | `Negosiasi` / `2026-09-01` | Tidak membuat seluruh stage history. |
| Row yang sama | `outlet` / `nilai_tahunan` | `35` / `147000000` | Planned outlets dan potential annual IDR snapshot. |
| Row yang sama | `status` | `Terbuka` | Snapshot status. |
| `crm_accounts[account_id=P04]` | `nama` / `tipe` | `Nirwana Hotel & Resto` / `prospek` | Current account metadata. |
| Row account yang sama | `champion_contact_id` | empty | Jangan menunjuk K065 sebagai champion otomatis. |

Tidak ada native decision rows yang mempunyai `deal_id=DL-004`, tidak ada billing contract account P04, dan tidak ada daily usage account P04 pada file yang diperiksa. Ini absence pada sumber tersebut; tidak mengklaim semua komunikasi/izin dunia nyata tidak pernah ada.

### 4.2 Tiga interaction records native

| Source ID | Tanggal | Tipe native | Participants/addresses native | Claim excerpt persis dari `isi` |
|---|---|---|---|---|
| `I0284` | 2026-08-10 | `catatan_meeting` | `peserta=K065;E06`, `dari=bagus@kasirnusa.id` | `35 outlet resto dan hotel. Kontak: Bu Yuli (Purchasing).` |
| `I0314` | 2026-08-28 | `catatan_meeting` | `peserta=K065;E06`, `dari=bagus@kasirnusa.id` | `Demo berjalan baik. Masuk tahap negosiasi harga.` |
| `I0335` | 2026-09-22 | `email` | `dari=yuli.astuti@nirwanahotel.co.id`, `ke=bagus@kasirnusa.id` | `Pak Bagus, Direktur Utama kami minta rekomendasi dari pengguna yang mirip dengan kami sebelum tanda tangan. Kami tunda dulu sampai ada referensi.` |

Ketiganya mempunyai native `account_id=P04`; ketiganya **tidak mempunyai field `deal_id`**. Shared account dengan DL-004 adalah candidate context linkage untuk slice, bukan bukti bahwa native interaction→deal edge sudah ada. Ketiga `membalas_id` kosong; jangan membuat reply thread dari kemiripan subjek.

I0314 merupakan bukti kalimat negosiasi pada 2026-08-28. Itu tidak otomatis menetapkan enum stage historis `Negosiasi`: stage anchor CRM 2026-09-01 dan text claim disimpan terpisah sampai aturan stage disepakati. Historical `stage` mengikuti limitation draft.

I0335 mendukung pernyataan **pengirim menyebut penundaan sampai ada referensi**. Tidak membuktikan pelanggan reference tertentu sudah consent, referensi sudah diberikan, atau identitas Direktur Utama dalam pesan pasti K028.

### 4.3 Participant support dan employment

| Locator source | Nilai relevant | Yang didukung / limitation |
|---|---|---|
| `employees[employee_id=E06]` | Bagus Prakoso, Sales Executive, `bagus@kasirnusa.id` | Native meeting participant E06; email lookup unique saat snapshot. Tidak ada employee role history. |
| `crm_contacts[contact_id=K065]` | Yuli Astuti, `yuli.astuti@nirwanahotel.co.id`, account P04, Purchasing Manager | Native meeting contact K065; sender email I0335 exact current-email candidate. |
| History tuple `K065/P04/Nirwana Hotel & Resto/Purchasing Manager/2021-06-01/(empty)` | Open current employment | Mendukung account/job interval pada tanggal tiga interaksi; alias email validity tetap bukan field history. |
| `crm_contacts[contact_id=K028]` | Hartono Gunawan, account P04, Direktur Utama | Current role snapshot, bukan participant tersurat di tiga records. |
| History tuple `K028/P04/Nirwana Hotel & Resto/Direktur Utama/2020-01-01/(empty)` | Employment sejak 2020 | Mendukung kandidat role matching, bukan otomatis mengidentifikasi kata “Direktur Utama kami” sebagai K028 verified. |
| History K028 sebelumnya | PT Sentosa Abadi Group, General Manager, 2015-01-01 … 2019-11-30, account kosong | Organisasi luar bersumber, tidak invent account ID bisnis. |

History tuples ditampilkan untuk source inspection. Native row ID tidak tersedia; physical source lines/checksums di audit JSON membantu menemukan row, tetapi stable derived-ID rule tetap draft.

### 4.4 Expected date filter observations

Ini **hasil filter atas field native `tanggal` pada account P04**, bukan hasil adapter/database/Graph API. Seluruh hari kalender inclusive pada date-only source; tidak menentukan timezone timestamp yang tidak ada di source.

| `as_of` untuk observasi | Interaction IDs yang bertanggal ≤ tanggal itu | Yang tidak boleh bocor dari record future |
|---|---|---|
| 2026-09-01 | `I0284`, `I0314` | ID/body I0335 belum visible; tidak boleh mengutip penundaan reference dari pesan 22 September. |
| 2026-09-22 | `I0284`, `I0314`, `I0335` | Jangan menambahkan consent/approval/outcome yang tidak didukung source. |
| 2026-10-01 | `I0284`, `I0314`, `I0335` | CRM snapshot available; tidak mengubah account-only linkage menjadi native deal edge. |

Untuk tanggal sebelum snapshot, stage/owner/ACV/outlet value belum diketahui historis dari snapshot row semata. Tiga date-filter observations ini cukup untuk expected fixture assertions setelah interface disepakati; belum membuktikan ID/edge/provenance graph maupun akses sudah benar.

## 5. Bahan review preseden tindakan — belum katalog sah

Beberapa source records punya konteks yang bisa ditinjau O1/O2 setelah rules disepakati:

| Native record | Source context | Limitation wajib |
|---|---|---|
| `decision_log D-2025-02` | C23, DL-006; diskon 20%, `Ditolak` | Rejection pada deal lain, bukan rejection/approval P02. Field `alasan` punya retrospective outcome; jangan memakai seluruh quote pada tanggal keputusan lama. |
| `decision_log D-2025-06` | C23, DL-007; `Paket Starter tanpa diskon, pilot 6 outlet`, `Disetujui` | Approval opsi paket pada deal lain; teks menang Sep 2025 tidak berlaku pada 12 Agustus. Tidak klaim pilot caused Won atau sudah completed. |
| `decision_log D-2025-09` | C24; diskon 12%, alasan `Referensi publik untuk industri F&B` | Account-only decision; bukan bukti consent universal, bukan reference suitability P04, tidak punya link interaction native. |
| `interaction I0334` | P03, 2026-09-21; buyer minta referensi pelanggan apotek | Buyer request berbeda dari Sales sudah menyediakan/reference telah consent. Tidak otomatis ActionTemplate executed. |
| `interaction I0335` | P04, 2026-09-22; meminta rekomendasi pengguna mirip | Bukti kebutuhan/gate dalam quote, belum tindakan Sales completed atau approval target. |

Belum ada klaim “P04 punya ≥2 kandidat valid” dari audit ini. Valid template/occurrence, relevance, eligibility compare, analog permissions dan consent harus lolos kontrak. Jumlah kandidat valid tetap bisa 0/1; tidak membuat opsi agar comparison berjalan.

## 6. Kasus handoff nyata untuk Orang 2

Kasus-kasus berikut bersumber dari file kompetisi dan cocok sebagai bahan assertions; nama response/error/enum tetap mengikuti kontrak yang nanti disepakati:

| Case | Source assertions | Peran O1 / O2 berikutnya |
|---|---|---|
| P04 date switch | Native account P04 interaction sets berubah 2→3 antara 1 dan 22 September | O1 menjaga refs/date-safe projection; O2 menjaga atomic panel/cache context. |
| P05 unknown | DL-005/P05 ada, native account interactions 0 | O1 empty source context; O2 tidak mengubah menjadi skor 0/negative intent. |
| Rina transition | K017 current P01, history C01 lalu P01, historical address unmatched | O1 ambiguity/source support; O2 tidak mengklaim current role berlaku pada seluruh history. |
| P02 request without native approval link | I0348 meminta/usul diskon20%; tidak ada decision native deal_id DL-002 | O1 request quote dan absence native link; O2 warning policy >10%, tidak membuat approval dari preseden C23. |
| C23 multiple deals | DL-006 dan DL-007 satu account; decision native links terpisah | O1 tidak duplicate account-only evidence ke semua deals; O2 tidak membandingkan sebagai izin P02. |
| Retrospective decision fields | D-2025-06 approval dated Agustus memuat winner text September | O1 split/limit claim setelah kontrak; O2 tidak membawa outcome future ke JEV/Copilot input historical. |

## 7. Pekerjaan berikutnya setelah persetujuan kontrak

Artefak ini menyediakan observations dan source locators untuk mempercepat review O2. Ketika kontrak inti disetujui, O1 dapat menetapkan canonical IDs, field/span evidence units, agreed temporal/linkage rules, domain structs dan adapter query untuk database statis. O2 dapat membangun DTO/service terhadap test-only stubs lalu menggantinya dengan adapter O1.

Sampai saat itu: tidak mengubah shared models/interface/routes/services, tidak menulis migration/database, tidak menambahkan pipeline, dan tidak mengaku source-audit JSON merupakan output repository production. Koneksi Aura, database contents, live API/JEV/Copilot dan performance belum diperiksa oleh mapping ini.
