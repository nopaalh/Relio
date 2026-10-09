# Relio — Keputusan Final v1.0 dan Pertanyaan Teknis Terbuka

**Tanggal acuan PRD:** 9 Oktober 2026  
**Acuan produk:** [PRD Final MVP v1.0](../../Relio_PRD_Final_MVP_v1.0_Agent_Ready_ID.md), menggantikan PRD/spec sebelumnya.  
**Dokumen terkait:** [Arsitektur backend](architecture.md) dan [Rencana pengembangan dua orang](development-plan.md).  
**Status:** keputusan produk yang diberi label **Final v1.0** berasal dari PRD. Semua **saran**, alternatif implementasi, dan **owner usulan** di sini belum disetujui; tidak otomatis memfinalkan vendor, framework, auth, atau angka batas baru.

## 1. Tujuan dan cara mengambil keputusan

PRD v1.0 adalah source of truth produk, bukan bukti bahwa implementasi atau integrasi sudah berjalan. Q01–Q24 dipertahankan untuk cross-reference; Q25–Q30 menutup action comparison dan handoff dalam pembagian kerja dua orang.

**Pembagian yang diberikan user:** **O1 / Orang 1 — context graph dan data**; **O2 / Orang 2 — aplikasi backend**. **Joint** berarti koordinasi O1+O2, bukan penambahan anggota tim. Owner pada register adalah rekomendasi untuk keputusan/handoff, bukan penugasan final. Pembagian ini belum menentukan pelaksana UI, E2E, dan evaluasi P0; lihat Q29. Pembagian nama pada PRD tidak otomatis dipakai untuk rencana dua orang ini.

Setiap pertanyaan membedakan keputusan final PRD dari detail terbuka, lalu memberi usulan beserta **alasan, risiko, dan tahap saat dibutuhkan**. Tahap merujuk fase PRD §11 atau dependency pekerjaan, bukan estimasi hari maupun janji timeline.

**Prinsip:** kerjakan parsing/domain/test yang sudah jelas; blokir hanya bagian yang bergantung pada keputusan atau bukti yang belum tersedia. Perubahan keputusan final harus melalui change request, bukan jawaban diam-diam di register ini.

### Keputusan v1.0 yang tidak ditanyakan ulang

- P0 mencakup graph temporal interaktif, timeline/evidence, assessment, **JEV asli**, perbandingan tindakan historis, Copilot read-only, serta pengujian/evaluasi A/B/C. Tidak ada execution tindakan, approval workflow baru, atau klaim peluang Won/uplift kausal.
- Rekonstruksi memakai `event_at <= as_of`, maksimum dataset `2026-10-01`; current snapshot tidak menjadi riwayat fiktif. Field tanpa sejarah diberi `snapshot_only`/batasan atau disembunyikan; `stage_as_of = unknown` sebelum snapshot bila histori tidak tersedia.
- Daya tarik tetap formula 50/50 PRD; denominator ACV adalah **maksimum ACV prospek aktif pada snapshot**, bukan pelanggan existing atau subset filter UI.
- Urgency memakai nilai dan bobot tetap PRD (`.55/.25/.20`, only-known), dengan rentang deadline mendekat **7 hari**. Ini default engineering final untuk v1.0, bukan temuan jurnal; boundary kalender dan lifecycle bukti masih terbuka.
- Coverage memakai **lima dimensi tetap**: `deal_record`, `stakeholder_context`, `meaningful_interaction`, `commercial_terms`, `decision_or_gate_evidence`. Kriteria known masih terbuka, bukan daftar/denominator baru.
- Readiness memiliki lima level **0..4**: `no_evidence_of_intent`, `initial_interest`, `active_evaluation`, `commercial_discussion`, `explicit_commitment_or_final_gate`; `readiness_100 = round(25 * jev_score)`. Score adalah ekspektasi tertimbang; cutoff label dan kecukupan bukti masih terbuka. Bukti tidak cukup → `null/insufficient_evidence`, bukan otomatis level 0.
- ActionTemplate membutuhkan ActionOccurrence berbukti; pengguna memilih **2–4 opsi**; Score per opsi dan Choice antaropsi terpisah, bukan probabilitas closing. Consent unknown, approval >10%, serta `requires_validation` tidak boleh disamarkan sebagai izin eksekusi.
- PRD menyebut endpoint/payload TypeSafe dan `jev-latest`; itu **kontrak acuan, bukan integrasi verified**. Cache v1 berbasis **hash input bundle + versi pertanyaan + model + snapshot**. POC live tetap blocker penerimaan.
- Stack **default bersyarat untuk repo kosong** adalah Go/Gin, Python ETL, Neo4j Community via Docker bila feasible; bukan migrasi wajib. Scaffold aktual memakai stdlib `net/http`; Go importer bukan default PRD. Multi-tenant production auth di luar scope, tetapi proteksi demo/data/endpoint berbayar tetap diperlukan.

### Prioritas

| Prioritas | Makna |
|---|---|
| **A — Fondasi** | Putuskan sebelum schema/import/context query dikunci. |
| **B — Assessment** | Putuskan sebelum skor/model dipakai sebagai kontrak produk. |
| **C — Integrasi UI/API** | Putuskan sebelum frontend bergantung pada DTO dan perilaku response. |
| **D — Rilis/operasi** | Putuskan sebelum deployment dibuka; sebagian harus diverifikasi sejak awal. |

### Register singkat

**Status register:** **Partially clarified** = prinsip/kontrak sudah dijelaskan PRD v1.0, tetapi detail teknis atau verifikasi masih terbuka; **Open** = pilihan operasional belum ditetapkan. Keduanya bukan tanda implementasi lulus. **Owner usulan** mengikuti O1/O2/joint di atas.

| ID | Detail yang masih terbuka | Status | Prioritas | Owner usulan | Tahap saat dibutuhkan |
|---|---|---|---|---|---|
| Q01 | Storage dan deployment terhadap baseline bersyarat PRD | Partially clarified | A | Joint: O1 storage, O2 runtime | Sebelum adapter/migration, fase 0–1. |
| Q02 | stdlib vs Gin dan jalur Python ETL | Partially clarified | A | Joint: O1 ETL, O2 HTTP | Sebelum dependency dan handoff importer, fase 0–1. |
| Q03 | Boundary `as_of`, timezone dan interval | Partially clarified | A | Joint | Sebelum query temporal, fase 1–2. |
| Q04 | Mapping event-time/knowledge-time/provenance | Partially clarified | A | O1 | Sebelum mapping sumber, fase 1. |
| Q05 | Ketersediaan field historis CRM | Partially clarified | A | Joint: O1 provenance, O2 DTO | Sebelum historical summary, fase 1–2. |
| Q06 | Namespace identitas dan crosswalk email | Partially clarified | A | O1 | Sebelum identity graph, fase 1. |
| Q07 | Stable ID, dedup, splitting dan revision | Partially clarified | A | O1 | Sebelum persistensi/reimport, fase 1. |
| Q08 | Usage parsial, missing/zero dan coverage | Partially clarified | A | O1 | Sebelum usage aggregate, fase 1. |
| Q09 | Schema fisik graph dan konflik klaim | Partially clarified | A | Joint: O1 schema, O2 consumer | Sebelum traversal/DTO, fase 1–2. |
| Q10 | Link request–approval–application/resolution | Partially clarified | A | Joint | Sebelum policy/gate assessment, fase 1–3. |
| Q11 | Kriteria prospek aktif, versi denominator dan histori | Partially clarified | B | Joint | Sebelum attractiveness, fase 2–3. |
| Q12 | Boundary deadline 7 hari dan lifecycle multi-signal | Partially clarified | B | Joint | Sebelum urgency/gold cases, fase 2–3. |
| Q13 | Kriteria known lima dimensi tetap | Partially clarified | B | Joint | Sebelum coverage, fase 2–3. |
| Q14 | Cutoff label dari weighted Score dan konflik historis | Partially clarified | B | Joint | Sebelum readiness/evaluasi, fase 3. |
| Q15 | POC live JEV, respons aktual, akses dan limit | Partially clarified | D, cek awal | O2; O1 bundle bukti | Mulai fase 0, blocker sign-off fase 3/MVP. |
| Q16 | Validator schema/evidence dan otoritas model | Partially clarified | B | Joint: O1 evidence, O2 validator | Sebelum assessment ditampilkan, fase 3–4. |
| Q17 | Provider Copilot dan retrieval | Open | B/C | Joint: O1 retrieval, O2 provider | Sebelum integrasi Copilot, fase 6. |
| Q18 | DTO/error dan detail kontrak action/API | Partially clarified | C | O2; review O1 dan owner UI Q29 | Sebelum consumer API, fase 2–5. |
| Q19 | Evidence snapshot-safe dan scope akses | Partially clarified | A/C | Joint | Sebelum evidence API, fase 2. |
| Q20 | Sync/async dan invalidation cache v1 | Partially clarified | B/C | O2 | Sebelum runtime assessment/compare, fase 3–4. |
| Q21 | Proteksi demo dan scope akses minimum | Partially clarified | D, desain awal | O2; review joint | Sebelum exposure deployment, fase 0/7. |
| Q22 | Budget, privacy, secret dan log | Open | D, cek awal | O2; O1 minimisasi data | Sebelum external model/exposure, fase 0/3. |
| Q23 | Gold cases, environment dan bukti acceptance | Partially clarified | B/C | Joint; pelaksana UI/E2E via Q29 | Mulai fase 1, sebelum sign-off fase 7. |
| Q24 | Distribusi data, publish, config dan readiness | Partially clarified | A/D | Joint: O1 data publish, O2 runtime | Sebelum import/deployment, fase 0–2/7. |
| Q25 | ActionTemplate/Occurrence, kategori dan dedup | Partially clarified | A/B | O1; review O2 | Sebelum katalog/handoff, fase 1/4. |
| Q26 | Relevance/eligibility candidates dan precedent cutoff | Partially clarified | B/C | Joint: O1 candidates, O2 policy API | Sebelum action-candidates/compare, fase 4. |
| Q27 | Rank Score vs Choice, ties, partial dan set/cache | Partially clarified | B/C | O2; review joint | Sebelum compare/UI, fase 4–5. |
| Q28 | Feasibility/consent dan requires_validation | Partially clarified | B/C | Joint | Sebelum kandidat/ranking ditampilkan, fase 4–5. |
| Q29 | Pelaksana UI/E2E dan evaluasi A/B/C | Partially clarified | C/D | Joint; pelaksana belum ditetapkan | Sepakati sebelum handoff UI, fase 0/2; terima fase 5–7. |
| Q30 | Handoff versi/data/provider dan ownership split | Partially clarified | A/C | Joint | Sebelum pekerjaan O1/O2 dikonsumsi silang, fase 0–2. |

## 2. Fondasi data, storage dan temporal

### Q01: Database/graph engine dan deployment apa yang dipilih?

**Pertanyaan:** Apakah baseline Neo4j Community via Docker feasible untuk tim, atau ada storage graph bertipe yang sudah memenuhi traversal temporal? Bagaimana API dan proses ETL dideploy tanpa sinkronisasi storage ganda?

**Final v1.0:** graph deal-centric adalah model relasi utama; untuk repo kosong PRD memberi default bersyarat Neo4j Community via Docker bila feasible. Storage alternatif harus mempertahankan typed graph, provenance, dan traversal temporal; tabel agregat tunggal tidak memenuhi kontrak.

**Detail terbuka:** engine/deployment aktual, indeks, cara publish versi dan operasi lokal belum diputuskan. Default PRD bukan bukti adapter sudah ada.

**Saran (usulan):** gunakan **modular monolith API** dengan proses ETL eksplisit. Uji baseline Neo4j terlebih dahulu; pertahankan storage setara yang sudah terbukti bila tersedia. PostgreSQL/Supabase dengan typed nodes/edges hanya alternatif yang perlu justifikasi dan spike, bukan pilihan final. Jangan menambah dua database graph atau memilih MongoDB hanya karena input JSONL.

**Alasan:** batas package sudah memberi modularitas tanpa biaya jaringan antarservice. Satu storage mengurangi sinkronisasi ganda. Format input tidak menentukan engine yang paling cocok untuk integrity/temporal graph.

**Risiko jika ditebak:** migration dan query dirombak; graph berubah menjadi tabel datar; atau terlalu banyak waktu habis untuk dua database.

**Bukti keputusan yang disarankan:** prototipe satu deal dengan query `as_of`, focus event, source lookup, dan reimport konsisten. Catat kemampuan yang berhasil dan tradeoff aktual, bukan hanya preferensi vendor.

**Dibutuhkan:** sebelum adapter/migration pada fase 0–1. **Owner usulan:** joint; O1 membuktikan storage/query, O2 memeriksa deployment dan konsumsi API.

### Q02: Framework HTTP dan importer dijalankan bagaimana?

**Pertanyaan:** Apakah stdlib `net/http` scaffold dipertahankan atau ada alasan konkret beralih ke Gin? Bagaimana O1 menjalankan Python ETL, menerbitkan hasil, dan menyerahkan kontrak data kepada O2 tanpa menjadikan import bagian dari setiap restart API?

**Final v1.0:** untuk repo kosong baseline PRD ialah **Go/Gin + Python ETL**. Solusi repo yang stabil dan setara tidak wajib dimigrasi. Scaffold aktual memakai stdlib `net/http`; keberadaan scaffold belum membuktikan seluruh P0 berfungsi.

**Detail terbuka:** pilihan framework aktual, packaging/command ETL, dependency Python, boundary output/provider dan mekanisme eksekusi belum disepakati. **Go importer bukan default PRD**; memilihnya memerlukan alasan eksplisit, bukan asumsi satu toolchain.

**Saran (usulan):** pertahankan stdlib selama memenuhi middleware/kontrak API yang dibutuhkan. O1 mulai dari arah Python ETL PRD dengan pemrosesan usage terbatas memori; O2 mengonsumsi data yang telah dipublish melalui kontrak Q30. Eksekusi import eksplisit dan repeatable, bukan otomatis setiap startup atau endpoint upload publik. Go importer hanya alternatif jika bukti biaya/kemudahan tim membenarkannya. Nama command/runtime belum final.

Orchestration model dapat mengikuti `services/`, adapter storage mengikuti `repository/`, dengan interface kecil yang membatasi dependency vendor. Penempatan ini usulan untuk mengikuti scaffold, bukan kontrak paket baru.

**Alasan:** memisahkan tanggung jawab O1/O2 dan menghindari migrasi framework tanpa manfaat; arah Python konsisten dengan default PRD, sedangkan stdlib memanfaatkan scaffold yang ada.

**Risiko jika ditebak:** dua importer berbeda menghasilkan ID/schema berbeda, dependency ditambah tanpa kebutuhan, API restart mengimpor ulang, atau arah PRD diganti tanpa keputusan tercatat.

**Dibutuhkan:** sebelum dependency dan jalur ingest dikunci, fase 0–1. **Owner usulan:** joint; O1 ETL, O2 HTTP/runtime, boundary melalui Q30.

### Q03: Apa arti persis `as_of` dan batas waktunya?

**Pertanyaan:** Apakah formatnya `YYYY-MM-DD` atau RFC3339? Tanggal berarti awal hari atau seluruh hari? Zona waktu apa? Apakah `valid_to` inklusif? Bagaimana tanggal invalid, setelah cutoff, atau sebelum deal dibuat?

**Final v1.0:** cutoff maksimum dataset `2026-10-01`, filter event `event_at <= as_of`, dan snapshot yang sama untuk panel terkait.

**Detail terbuka:** boundary tanggal/timestamp, timezone, interval masa kerja, serta error untuk tanggal invalid/di luar batas belum rinci.

**Saran:** untuk MVP gunakan tanggal ISO `YYYY-MM-DD` yang mewakili **akhir hari bisnis Asia/Jakarta**; query internal dapat memakai batas eksklusif awal hari berikutnya. Default hanya digunakan bila parameter tidak diberikan; tanggal invalid/setelah cutoff ditolak, tidak diam-diam di-clamp. Gunakan interval internal `[valid_from, valid_to)`, dengan konversi eksplisit jika `selesai` pada sumber berarti tanggal terakhir menjabat.

Tanggal sebelum deal dibuat tidak boleh menampilkan deal seolah sudah ada; pilih kontrak error/empty snapshot secara eksplisit pada Q18.

**Alasan:** sumber yang dideskripsikan PRD banyak memakai tanggal; precision raw tetap perlu diaudit. Satu aturan boundary mencegah perbedaan antarendpoint dan bug di hari perpindahan kerja.

**Risiko jika ditebak:** event yang sama muncul di timeline tetapi hilang di graph, job baru dan lama aktif bersamaan, atau cutoff di-clamp tanpa diketahui pengguna.

**Dibutuhkan:** sebelum query temporal dan unit test boundary. Asia/Jakarta dan end-of-day adalah **usulan**, bukan aturan PRD yang sudah final.

### Q04: Apakah rekonstruksi berarti kejadian saat itu atau pengetahuan saat itu?

**Pertanyaan:** Apa pemetaan `observed_at`, `event_at`, `recorded_at`, dan ingestion time? Bagaimana menangani sumber yang diperbarui belakangan, serta tanggal future renewal yang telah dijadwalkan?

**Final v1.0:** bila sumber tidak memiliki `recorded_at`, gunakan `event_at` untuk rekonstruksi kejadian dan nyatakan bahwa waktu pengetahuan internal tidak dapat dipastikan. Edge menyimpan `observed_at` jika tersedia; ingestion time bukan bukti knowledge-time.

**Detail terbuka:** mapping waktu per sumber, arti persis `observed_at`, revisi sumber dan rencana masa depan yang sudah terdokumentasi belum rinci.

**Saran (usulan mapping):** terapkan baseline event-time PRD ketika knowledge-time tidak tersedia. Simpan timestamp sumber dan ingestion metadata terpisah; jangan mengganti `recorded_at` dengan waktu import. Definisikan `observed_at` per source mapping, nullable bila benar-benar tidak ada bukti waktunya. Labeli klaim yang berasal dari current snapshot sebagai terbatas untuk sejarah.

Tanggal renewal masa depan pada kontrak dapat ditampilkan sebagai **rencana yang terdokumentasi**, jika dasar keberadaan kontrak pada snapshot memadai; jangan membuat event renewal seolah telah berlangsung. Outcome/status yang baru diketahui belakangan tidak ditempelkan mundur ke kejadian lama.

**Alasan:** timestamp import tidak membuktikan kapan organisasi mengetahui suatu fakta. Peristiwa masa depan berbeda dari rencana masa depan yang sudah tercantum dalam dokumen.

**Risiko jika ditebak:** sistem mengklaim bitemporal audit yang tidak dimiliki data, atau melabeli outcome masa depan sebagai pengetahuan historis.

**Dibutuhkan:** sebelum mapping provenance dan desain historical evidence. Keterbatasan harus terlihat pada API/UX.

### Q05: Bagaimana field deal/current CRM direkonstruksi ke masa lampau?

**Pertanyaan:** Dari mana stage sebelumnya, owner sebelumnya, ACV/rencana outlet sebelumnya, champion sebelumnya, status penutupan, dan waktu perubahan masing-masing diperoleh?

**Final v1.0:** `crm_deals.stage` adalah kondisi snapshot, bukan riwayat lengkap. Jika sejarah tidak tersedia, tanggal sebelum snapshot memakai `stage_as_of = unknown`; current facts diberi `snapshot_only` atau disembunyikan pada mode historical, tidak diklaim diketahui di masa lampau.

**Detail terbuka:** availability/provenance per field, sumber perubahan owner/ACV/outlet/champion, dan pengaruh history ambigu terhadap skor/denominator perlu diaudit.

**Saran (usulan kontrak field):** gunakan bukti bertanggal bila ada, selain itu null dengan alasan/batasan historical. Jangan menyimpulkan stage lampau hanya dari `stage_sejak`, menganggap ACV/owner tetap sejak awal, atau menerapkan status Closed Won sebelum bukti penutupan. Pilih representasi field-level melalui Q18/Q30.

**Alasan:** lebih baik partial historical context yang jujur daripada playback lengkap tetapi fiktif. Ketidakpastian ini juga memengaruhi daftar active deals dan denominator ACV pada Q11.

**Risiko jika ditebak:** future leakage walau event sudah difilter; skor lampau dihitung dari nilai yang belum tentu berlaku saat itu.

**Dibutuhkan:** sebelum historical summary, overview dan assessment. Tentukan field-level provenance/availability, bukan satu flag umum untuk seluruh deal.

### Q06: Apa identitas canonical dan bagaimana email lama diselesaikan?

**Pertanyaan:** Apakah Contact/Employee menjadi satu node `Person` dengan namespace? Bagaimana menangani email lama, dua kandidat, organisasi tanpa account ID, dan masa kerja yang bertumpang tindih?

**Final v1.0:** `Person` memiliki subtipe `Employee`/`Contact`; identitas email historis diresolve dengan bukti dan masa kerja **pada tanggal event**, bukan spekulasi nama atau perusahaan terkini.

**Detail terbuka:** namespace/crosswalk, ambiguity dan overlapping employment belum rinci. Riwayat kerja tidak dengan sendirinya membuktikan alamat email lama; field sumber aktual perlu diaudit.

**Saran:** gunakan namespace terpisah, misalnya person-contact dan person-employee, dengan ID asal sebagai kunci. Utamakan ID peserta eksplisit. Untuk email, bangun crosswalk berbukti dan bertanggal bila sumber mendukung; employment history membantu membatasi kandidat, tetapi tidak membuktikan alamat email sendiri. Multiple plausible matches tetap `ambiguous`; jangan auto-merge dari nama/domain saja.

Organisasi tanpa `account_id` tidak dipaksa menjadi pelanggan. Normalisasi email secukupnya tanpa aturan spekulatif seperti menghapus semua titik atau menganggap alias pasti orang yang sama.

**Alasan:** histori relasi orang adalah nilai utama context graph; kesalahan identity menyebar ke keputusan, ownership, dan klaim Copilot.

**Risiko jika ditebak:** stakeholder orang lain ditampilkan sebagai pemberi approval atau kontak lama terhubung ke perusahaan terkini.

**Dibutuhkan:** sebelum identity import/traversal. Format ID di atas adalah contoh desain, bukan identifier produksi yang sudah disepakati.

### Q07: Bagaimana stable ID, splitting event, deduplication, dan revision bekerja?

**Pertanyaan:** Satu interaksi menjadi satu event atau beberapa? Bagaimana ID tetap stabil setelah reimport? Jika isi sumber/rule ekstraksi berubah, apakah ID diganti atau diberi revision? Bagaimana Event dan Decision tidak menggandakan satu fakta?

**Final v1.0:** ID sumber stabil, ETL idempotent, reimport identik menjaga ID/jumlah node/edge, dan seluruh error row-level dilaporkan.

**Detail terbuka:** algoritma derived ID, splitting klaim, dedup lintas Interaction/Decision/ActionOccurrence, dan revision/extraction version; action dedup dirinci di Q25.

**Saran:** pisahkan ID sumber dari event/claim ID. Gunakan source record ID yang tersedia; untuk baris tanpa ID gunakan canonical composite key yang divalidasi unik. ID event diturunkan dari source identity dan identitas klaim yang deterministik; simpan revision/extraction version terpisah. Jangan memakai nomor baris saja atau ringkasan LLM sebagai kunci. Event keputusan menunjuk Decision sumber yang sama.

Simpan event dasar interaksi; pecah menjadi klaim/request tambahan hanya bila bukti dan rule jelas. Reimport identik harus idempotent; perubahan isi harus dapat diaudit, bukan diam-diam menambah event baru identik secara makna.

**Alasan:** highlight UI, citations, cache, dan test membutuhkan ID stabil. Baris berpindah urutan atau parafrasa ringkasan tidak boleh memutus semua link.

**Risiko jika ditebak:** graph terduplikasi, urgency berlipat, kutipan usang, dan persetujuan terhitung dua kali.

**Dibutuhkan:** sebelum schema persistensi event/edge dan reimport test.

### Q08: Bagaimana agregasi usage untuk snapshot parsial dan data kosong?

**Pertanyaan:** Pada tengah bulan, bagaimana transaksi dan jumlah outlet aktif dihitung? Apa definisi aktif: punya row, transaksi `>0`, atau status lain? Bagaimana missing vs zero, duplicate rows, offline metrics, dan feature usage bulanan?

**Final v1.0:** usage diagregasi per akun/bulan hanya untuk pelanggan existing/pembanding, bukan usage atau kesehatan finansial prospek; transaksi harian tidak menjadi node individual.

**Detail terbuka:** coverage, snapshot parsial, definisi outlet aktif dan pemisahan missing/zero belum rinci.

**Saran:** filter data harian berdasarkan cutoff sebelum agregasi; simpan periode yang benar-benar tercakup dan status partial. Bedakan jumlah outlet dengan data tercatat dari outlet dengan transaksi `>0`; pilih metrik utama secara eksplisit. Missing row bukan transaksi nol. Validasi duplicate key sumber dan mapping account/outlet sebelum penjumlahan.

Untuk feature usage yang hanya berupa satu nilai bulanan, jangan membagi rata nilai bulan penuh ke tanggal lampau. Gunakan periode lengkap yang sudah tersedia atau tampilkan tidak tersedia untuk cutoff parsial. Jangan menambahkan `transaksi_offline_tersinkron` ke `jumlah_transaksi` tanpa memverifikasi apakah keduanya overlap.

**Alasan:** penggunaan pelanggan analog harus dapat dijelaskan; agregat penuh yang ditempel pada tengah bulan merupakan kebocoran masa depan.

**Risiko jika ditebak:** volume transaksi double counted, missing menjadi nol, dan analog terlihat lebih sehat/aktif daripada bukti sebenarnya.

**Dibutuhkan:** sebelum materialisasi usage aggregate dan retrieval analog.

### Q09: Bagaimana schema graph rinci, scope klaim, dan konflik direpresentasikan?

**Pertanyaan:** Bagaimana schema fisik node/edge minimum PRD, termasuk Evidence dan ActionOccurrence, diimplementasikan? Bagaimana Assessment/Buyer request dihubungkan tanpa mencampur fakta dengan evaluasi? Apa arah relasi Event→Interaction/Decision, Contract→Decision, reply thread dan template→occurrence?

**Final v1.0:** node/edge minimum pada PRD §3 harus terwakili, Evidence mempunyai provenance, `verification_state` memakai `verified/inferred/ambiguous`, dan `SIMILAR_PRECEDENT` tetap inferred. Representasi Graph API tidak boleh kehilangan relasi/waktu.

**Detail terbuka:** schema fisik, kriteria verification, relasi tambahan, representasi assessment/konflik dan integrasi ActionTemplate Q25 belum lengkap.

**Saran (usulan):** buat registry versi untuk node, typed edge, claim kind, verification state, dan business state. Pertahankan Evidence sebagai node model graph sesuai PRD dengan source record yang addressable lewat ID; detail persistensi dapat mengikuti engine Q01. Tambahkan relasi sumber/request/kontrak/thread/template hanya setelah nama, arah dan provenance disepakati.

Pertahankan klaim yang bertentangan beserta sumber; jangan memilih salah satu secara diam-diam. Scope account-only tetap account-only. Inferential precedent membawa alasan/ketidakpastian dan tidak menjadi edge approval. `Observed` berarti ada pada sumber; `verified` berarti lolos kriteria validasi yang disepakati, bukan kepastian universal.

**Alasan:** visual graph, traversal dan validator perlu kosakata konsisten. Satu field `status` untuk semuanya akan mencampur kualitas data dan keputusan bisnis.

**Risiko jika ditebak:** node/edge tidak bisa di-query konsisten, model assessment terlihat sebagai fakta, dan account decision dipromosikan menjadi izin deal.

**Dibutuhkan:** sebelum schema/traversal dan DTO graph dikunci.

### Q10: Bagaimana request, approval, resolution, dan application dihubungkan?

**Pertanyaan:** Apa request ID/cakupan persetujuan? Apakah approval mengizinkan nominal tertentu, periode, syarat pembayaran, atau satu kontrak? Bagaimana request baru, supersession, gate resolved, dan otoritas VP pada waktu keputusan dibuktikan?

**Final v1.0:** `requested ≠ proposed ≠ approved ≠ rejected ≠ applied`; diskon >10% membutuhkan VP Sales. Keputusan kasus lain bukan approval deal ini; status/angka tidak diwariskan ke permintaan baru tanpa bukti terkait.

**Detail terbuka:** kelengkapan link request→decision→contract dan bukti otoritas pada tanggal keputusan harus diaudit; PRD tidak menjamin request linkage atau histori jabatan universal. Batas feasibility/consent untuk compare dijelaskan Q28.

**Saran:** gunakan link eksplisit atau bukti tekstual spesifik untuk request→decision→contract. Bila hanya account/nilai mirip, jangan otomatis menghubungkan sebagai approval/application request tersebut. Pisahkan request baru dari revisi; resolution membutuhkan bukti yang menunjuk requirement yang sama. Tampilkan role/authority yang diketahui dan keterbatasan histori, bukan mengarang jabatan lampau.

Jika perlu curated mapping untuk kasus ambigu, catat siapa meninjau, sumber, alasan, dan verification; mapping manual tidak menambahkan fakta baru ke sumber asli.

**Alasan:** kebijakan `>10%` membutuhkan approval dengan scope benar, bukan sekadar adanya satu approval historis pada akun.

**Risiko jika ditebak:** approval 10% dipakai untuk 14%, diskon applied di kontrak lain dianggap mengizinkan deal saat ini, atau gate dianggap resolved tanpa bukti.

**Dibutuhkan:** sebelum discount history dan fitur urgency purchase gate. Tidak menambahkan approval workflow MVP.

## 3. Rubrik assessment dan integrasi model

### Q11: Bagaimana kriteria prospek aktif, versi denominator ACV dan history ambigu?

**Final v1.0:** `acv_relative = clamp(100 * acv / max(acv prospek aktif pada snapshot), 0, 100)`; denominator ditampilkan. Daya tarik memakai 50/50 ACV relatif dan strategic outlets bila keduanya diketahui. Populasi **prospek aktif pada snapshot** sudah final, bukan pilihan antara semua pelanggan/lima baris UI.

**Detail terbuka / pertanyaan:** field/status apa membuktikan aktif, bagaimana deal dengan ACV invalid/unknown ditangani, apa versi populasi yang dipublish, dan bagaimana snapshot lampau dinilai bila status/value tidak punya history? Bagaimana denominator kosong/nol dan pembulatan selain formula yang telah ditentukan?

**Saran (usulan):** O1 menerbitkan population IDs, denominator, checksum/versi snapshot dan alasan eksklusi dari audit; O2 memakai nilai yang sama, tidak menghitung ulang atas hasil pencarian UI. Jangan otomatis menyamakan populasi aktif dengan P01–P05 sebelum status diaudit. Denominator historis hanya tersedia bila status/nilai pada cutoff berbukti; selain itu score null dengan alasan. Nol/kosong tidak diubah menjadi penyebut buatan; precision dan presentation rounding disepakati di Q18.

**Alasan:** populasi yang addressable menjaga reproduksibilitas antar-O1/O2 dan menjelaskan mengapa denominator tersedia atau tidak.

**Risiko jika ditebak:** filter mengubah skor, status Closed Won masa depan bocor ke denominator historis, atau unknown diubah menjadi nol.

**Dibutuhkan:** sebelum attractiveness pada fase 2–3. **Owner usulan:** joint; O1 audit populasi/history, O2 kalkulasi/DTO. Formula tidak diubah tanpa change request.

### Q12: Bagaimana boundary deadline 7 hari dan lifecycle urgency multi-signal?

**Final v1.0:** deadline terlewat `100`, mendekat dalam **7 hari** `75`, lebih jauh `25`, tanpa tenggat diketahui `null`. Purchase gate memakai `100/40/0/null`, unresolved follow-up `75/0/null`; bobot `.55/.25/.20` hanya atas komponen known. Semua null → null; stage age bukan bobot otomatis. Tujuh hari adalah default engineering final v1.0, bukan pertanyaan threshold baru atau temuan jurnal.

**Detail terbuka / pertanyaan:** apakah boundary tepat hari ke-7 dan jatuh tempo hari ini memakai tanggal bisnis atau timestamp, mengikuti timezone Q03? Apa bukti minimum gate/follow-up known/resolved? Bagaimana beberapa deadline/gate, revisi request dan history ambigu diagregasi tanpa double count?

**Saran (usulan detail):** dokumentasikan boundary inklusif hari snapshot sampai hari ke-7 sebagai tujuh hari kalender, dengan timezone yang disepakati Q03; jangan diam-diam menggantinya menjadi tujuh hari kerja. Purchase gate memerlukan kondisi eksplisit, bukan setiap permintaan referensi. Follow-up memerlukan janji/tugas tertulis; ketiadaan `membalas_id` sendiri bukan bukti tidak ditindaklanjuti. Kaitkan meaningful interaction ke kriteria coverage Q13.

Untuk banyak sinyal, uji severity tertinggi dari klaim valid/deduplicated per komponen, sambil menampilkan seluruh sinyal dan konflik. Gate `0` membutuhkan bukti resolution yang relevan; unknown tetap unknown. Rule agregasi/boundary ini usulan, tidak mengubah nilai/bobot PRD.

**Alasan:** kriteria lifecycle membuat angka final dapat diterapkan konsisten tanpa menciptakan SLA atau menghukum stage age.

**Risiko jika ditebak:** batas tanggal berbeda antarendpoint, request dihitung berulang, atau absence of record dianggap resolved/unanswered.

**Dibutuhkan:** sebelum urgency/gold cases fase 2–3. **Owner usulan:** joint; O1 klaim/lifecycle, O2 rule dan semantic-check JEV.

### Q13: Apa kriteria known untuk lima dimensi evidence yang sudah final?

**Final v1.0:** `required_dims` tetap **lima**: `deal_record`, `stakeholder_context`, `meaningful_interaction`, `commercial_terms`, `decision_or_gate_evidence`. Coverage berupa jumlah dimensi tersedia dari lima, misalnya `3/5`, bukan confidence closing. Daftar dan denominator tidak lagi terbuka.

**Detail terbuka / pertanyaan:** bukti minimum apa membuat masing-masing dimensi known pada cutoff? Apakah source tertaut tetapi ambigu/inferred/bertentangan cukup? Apa definisi meaningful interaction, dan bagaimana history/snapshot-only memengaruhi coverage? Bagaimana anotasi not-applicable dijelaskan tanpa mengubah denominator v1?

**Saran (usulan kriteria):** buat registry versi untuk kelima dimensi dengan rule evidence/scope/waktu dan contoh known/unknown. O1 menyerahkan evidence IDs serta availability; O2 menghitung coverage dengan rule yang sama. Dimensi hanya inferred atau conflicting tidak otomatis known; tampilkan keterbatasannya terpisah. Not-applicable tidak disimpulkan dari data kosong dan tidak menghapus dimensi dari denominator 5 tanpa change request. Komponen komersial/diskon dipetakan ke dimensi final, bukan menjadi dimensi keenam.

**Alasan:** kriteria known memungkinkan angka coverage diaudit tanpa membuka kembali scope rubrik final.

**Risiko jika ditebak:** denominator menyusut agar coverage tampak tinggi, missing dianggap known-zero, atau rasio disalahartikan sebagai akurasi/peluang Won.

**Dibutuhkan:** sebelum overview/assessment coverage fase 2–3. **Owner usulan:** joint; O1 availability/provenance, O2 registry/calculation.

### Q14: Bagaimana label readiness diturunkan dari weighted Score dan history ambigu?

**Final v1.0:** level 0 `no_evidence_of_intent`, 1 `initial_interest`, 2 `active_evaluation`, 3 `commercial_discussion`, 4 `explicit_commitment_or_final_gate`. `readiness_100 = round(25 * jev_score)`, dengan `jev_score` **ekspektasi tertimbang level 0..4**, sehingga dapat pecahan. Bukti tidak cukup memakai `null/insufficient_evidence` sebelum panggilan, bukan memaksa level 0.

**Detail terbuka / pertanyaan:** bagaimana `ordinal_label` dipilih dari distribusi/ekspektasi Score: level terdekat dengan cutoff tertentu, mode distribusi, atau aturan lain? Apa minimum evidence, cara menampilkan distribusi ambigu, dan aturan relevansi/recency bila stakeholder atau bukti historis bertentangan? PRD belum menentukan cutoff label dari score tertimbang.

**Saran (usulan):** pisahkan raw Score/distribusi, normalisasi numerik final, label tampilan dan status evidence. Uji kandidat mapping label pada gold cases; pilih/catat cutoff, boundary dan tie rule bersama, tidak diam-diam menganggap Score selalu integer atau rounding angka 0–100 sama dengan penentuan label. Rubrik memakai proposisi yang dapat dikutip untuk tiap level final. Jangan memilih maksimum sepanjang sejarah; periksa perubahan/penolakan terbaru yang valid pada cutoff dan tampilkan konflik yang belum terselesaikan. Status data/provider terpisah dari lima label ordinal.

**Alasan:** ketidaktahuan P05 bukan bukti tidak berminat; score pecahan membutuhkan mapping eksplisit agar label dan angka tidak saling menyesatkan.

**Risiko jika ditebak:** label berganti karena rounding tersembunyi, konflik dipromosikan menjadi commitment, atau readiness dibaca sebagai probabilitas Won.

**Dibutuhkan:** sebelum prompt/validator/DTO readiness fase 3. **Owner usulan:** joint; O1 evidence/history, O2 mapping/validator, review melalui gold cases Q23.

### Q15: Apakah API JEV asli dapat dipanggil dengan kebutuhan kita?

**Final v1.0:** PRD §4.5 menyebut `POST https://api.typesafe.ai/v1/systemone`, Bearer `TYPESAFE_API_KEY`, `model: "jev-latest"`, payload `state` dan map `questions` bertipe `choice/score/noul`. Ini **kontrak yang disebut PRD**, bukan integrasi verified atau bukti key/layanan sudah bisa dipakai.

**Detail terbuka / pertanyaan:** apakah credential/akses tersedia dan panggilan live berhasil dari runtime backend? Bagaimana bentuk aktual `answers.*`, metadata versi resolved di balik alias `jev-latest`, latency, limit, biaya, token usage dan error? Kesesuaian payload/response dengan dokumentasi serta kebutuhan Score/Choice/Noul tetap perlu diuji.

**Saran (usulan):** mulai authenticated POC pada fase 0 secara paralel: kirim bundle bukti kecil yang tervalidasi dan uji Choice/Score/Noul sebelum client penuh. Simpan response/metadata aman dan hasil validator Q16; cocokkan ke dokumentasi resmi yang dirujuk PRD, bukan method SDK rekaan. Jangan memakai placeholder contoh PRD sebagai fakta bisnis.

Jika akses belum tersedia, siapkan interface/adapter yang dapat diuji dan laporkan blocker; jangan menyebut mock sebagai API asli berhasil. Jangan otomatis mengganti JEV dengan provider lain karena kewajiban PRD tetap JEV.

**Alasan:** API eksternal/credential adalah risiko di luar kontrol kode dan syarat DoD; mengujinya terakhir berisiko menggagalkan demo.

**Risiko jika ditebak:** endpoint atau method rekaan, schema salah, credential terlambat, dan klaim integrasi palsu.

**Dibutuhkan:** mulai verifikasi fase 0; **POC live nyata tetap blocker** sebelum sign-off assessment/MVP. **Owner usulan:** O2 panggilan/client; O1 memasok bundle berbukti. Tidak ada tanggal penyelesaian yang diasumsikan.

### Q16: Kapan output model dinilai valid dan apa otoritasnya?

**Pertanyaan:** Cukup JSON schema-valid atau harus quote-supported? Siapa memastikan cited record memang mendukung klaim? Apa yang dilakukan jika enum benar tetapi evidence salah, unsupported ID, timestamp future, atau kontradiksi?

**Final v1.0:** validasi `answers.*` sesuai tipe, jumlah probabilitas Choice/Score sekitar 1, level/rentang sesuai rubrik dan evidence IDs sah. Penjelasan Relio berasal dari rubrik + pertanyaan + bukti + faktor deterministik, bukan otomatis narasi alasan internal JEV.

**Detail terbuka:** toleransi numerik, batas payload, dukungan quote/proposisi, metode review kontradiksi, makna `jev_confidence` dan representasi metadata yang tidak disediakan provider belum disepakati.

**Saran:** dua gerbang terpisah:

1. **Schema-valid:** field, enum, tipe, ukuran, dan format benar.
2. **Evidence-valid:** source IDs ada dalam retrieval, valid terhadap waktu/scope, dan kutipan/proposisi memiliki dukungan yang dapat diuji/ditinjau.

Simpan dua hasil validasi dan kategori error. Pisahkan hasil assessed dari `verification_state` graph; penilaian semantik tidak otomatis menjadi fakta `verified`. Unsupported output ditolak/diabstain; koreksi/retry dibatasi, bukan loop model tanpa batas. Field confidence/token usage/version yang tidak dikembalikan provider tidak diisi angka rekaan atau coverage; gunakan null/alasan dengan metadata yang benar-benar tersedia.

**Alasan:** structured output menjamin bentuk, bukan kebenaran. Model bisa mencantumkan ID nyata dengan makna yang salah.

**Risiko jika ditebak:** model membentuk approval palsu atau verified graph edge hanya karena JSON dapat diparse.

**Dibutuhkan:** sebelum menyimpan/menampilkan assessment dan semantic enrichment. Kualitas validator harus dievaluasi dengan kasus unsupported/contradicted.

### Q17: Provider Copilot dan strategi retrieval apa yang digunakan?

**Pertanyaan:** Provider pembentuk jawaban apa? Retrieval template query, graph traversal, full-text, embedding, atau kombinasi? Bagaimana pertanyaan lintas deal dan historical precedent dipilih? Siapa memeriksa graph paths? Apakah chat history digunakan ulang ketika deal, `as_of`, versi dataset, atau scope akses berubah?

**Final v1.0:** Copilot P0 read-only, klaim penting mempunyai evidence IDs yang ada pada bundle retrieval atau dinyatakan unknown/insufficient. LLM generatif boleh merangkai bahasa, tetapi JEV tetap untuk penilaian terstruktur.

**Detail terbuka:** provider, strategi retrieval, batas scope/paths dan policy chat history belum dipilih. JEV evaluator tidak otomatis API chat; vendor Copilot tidak final.

**Saran:** mulai dari graph traversal/source lookup terkontrol dan search sederhana atas bukti yang valid, lalu model menyusun jawaban terbatas. Retrieval dibatasi oleh scope, `as_of`, dataset version, dan akses. Tambahkan embedding hanya bila evaluasi pertanyaan baru menunjukkan kebutuhan nyata. Jangan menyerahkan arbitrary SQL/Cypher/write tool ke model.

Untuk precedent/analogue, definisikan alasan kemiripan dan batas kandidat yang dapat ditelusuri; jangan memilih hanya dari kemiripan nama atau menganggap outcome historis kausal. Validator memeriksa setiap node/edge path yang dikembalikan.

Untuk MVP, pisahkan context percakapan menurut scope deal/lintas deal, snapshot, dan versi data. Saat konteks berubah, reset model history atau validasi ulang semua history sebelum dipakai. Jawaban lama boleh tetap terlihat dengan label snapshot lamanya, tetapi bukan bukti otomatis bagi jawaban baru. Chat user yang menyebut approval masa depan juga bukan source evidence. Policy reset/persist final perlu disepakati.

**Alasan:** konteks utama hanya lima prospek dan ratusan interaksi; baseline sederhana lebih cepat diaudit daripada infrastruktur RAG besar.

**Risiko jika ditebak:** model melakukan query tak terkendali, mengutip kasus lain sebagai fakta deal, membawa approval September lewat history ketika `as_of` dimundurkan ke Agustus, atau stack vector ditambah tanpa kualitas jawaban yang terukur.

**Dibutuhkan:** sebelum integrasi Copilot; keputusan provider mempertimbangkan biaya/privacy pada Q22.

## 4. Kontrak API, frontend dan runtime assessment

### Q18: Seperti apa DTO, sorting, error, pagination dan bounded traversal?

**Pertanyaan:** Apakah response envelope sama? Field mana nullable? Bagaimana not-found vs bukti kosong vs assessment pending? Di mana `N/A` pada sort? Apa tie-break akhir, order event bertanggal sama, batas graph, dan respons focus event yang di luar snapshot?

**Final v1.0:** route internal PRD §8 mencakup action-candidates dan actions/compare, selain deals/graph/timeline/assessment/evidence/Copilot/health. Request compare memuat `as_of` dan `action_ids`; hasil mengungkap Score, Choice jika tersedia, rank, evidence/precedent IDs, policy flags dan unknowns. Overview mengurutkan urgency known tertinggi dan null ke grup **“Data belum memadai”**, bukan ranking rendah.

**Detail terbuka:** OpenAPI lengkap, nullable/status per field, error/HTTP codes, pagination, event ordering, batas graph serta metadata dataset belum final. Kontrak Relio bukan payload API JEV. Ranking/ties/partial compare dirinci Q27.

**Saran (usulan):** bekukan OpenAPI setelah domain/schema jelas dengan O1 dan pelaksana UI Q29. Gunakan null plus alasan/status untuk unknown, bukan `0`/empty string. Ikuti pengelompokan urgency/null final PRD; ACV menurun lalu stable `deal_id` dapat menjadi tie-break usulan. Sorting/filter daya tarik dan ACV tidak mengubah denominator Q11.

Events memakai sort stabil `event_at` lalu ID atau urutan sumber yang dapat dipertanggungjawabkan; jangan menganggap dua catatan harian memiliki urutan menit yang tidak tersedia. Batasi depth/node/edge graph dan ukuran evidence; response terpotong harus menyatakannya. Untuk focus event di luar scope/waktu, kembalikan error terstruktur atau clear focus eksplisit, bukan memasukkan event masa depan.

Response menyertakan snapshot serta version metadata untuk menghindari pencampuran dataset. Payload/response Copilot perlu menentukan context/session ID bila history digunakan dan mengikatnya ke akses/snapshot; jangan percaya `history` dari browser sebagai sumber terverifikasi. HTTP codes, error envelope, filter params, payload Copilot, dan pagination diputuskan bersama frontend sebelum implementasi konsumennya.

**Alasan:** DTO konsisten mempercepat integrasi dan menjaga makna unknown/pending. Bounded traversal melindungi performa dan akses.

**Risiko jika ditebak:** frontend menganggap model gagal sebagai low score, urutan tidak stabil, atau focus menyebabkan kebocoran masa depan.

**Dibutuhkan:** sebelum integrasi frontend. Lima baris overview tidak memerlukan pagination kompleks, tetapi event/evidence tetap perlu batas.

### Q19: Bagaimana evidence endpoint mengikuti snapshot dan akses?

**Pertanyaan:** Route evidence PRD tidak menyebut `as_of`; bagaimana server tahu cutoff/scope? Bukti disimpan per message atau seluruh thread? Bagaimana sources yang mutable dan data terlarang disanitasi?

**Final v1.0:** evidence API mengembalikan cuplikan sumber yang aman dengan provenance; seluruh panel konsisten dengan snapshot, evidence ID dari response harus dapat dibuka.

**Detail terbuka:** route evidence PRD tidak memuat cutoff, sehingga binding scope/waktu perlu ditetapkan. Raw thread dapat mengandung balasan masa depan meski graph sudah difilter.

**Saran:** ikat pengambilan evidence ke snapshot/context server yang terverifikasi atau tambahkan parameter `as_of` eksplisit sebagai perluasan kontrak yang disepakati. Evidence mengacu ke source record/message yang stabil, bukan otomatis seluruh thread. Ambil hanya bagian yang valid pada snapshot dan akses pengguna. Bila satu record mencampur informasi lintas waktu yang tidak dapat dipisahkan, jelaskan keterbatasan atau jangan tampilkan sebagai bukti historis valid.

Sanitasi excerpt untuk frontend; jangan mengekspos path filesystem absolut, raw HTML aktif, secret, atau metadata akses internal. Scope deal/account bukan otorisasi dengan sendirinya; auth policy tetap harus diuji.

**Alasan:** semua jalur menuju fakta, termasuk panel bukti dan Copilot, perlu aturan temporal/access yang sama.

**Risiko jika ditebak:** user mengubah tanggal graph ke Agustus tetapi membaca approval September melalui evidence ID.

**Dibutuhkan:** sebelum evidence API dan retrieval Copilot dipublikasikan.

### Q20: Assessment sync/async, cache dan failure state bagaimana?

**Pertanyaan:** Assessment/compare menunggu model, pending, atau membaca precompute? Berapa timeout/retry dan concurrency setelah latency/limit nyata diketahui? Bagaimana dedup, invalidation alias model, stale response dan failure per item dipaparkan?

**Final v1.0:** cache memakai **hash input bundle + versi pertanyaan + model + snapshot**; kegagalan JEV bukan skor palsu. State tanggal berubah atomik, tidak ada hasil lama diklaim untuk snapshot baru. Tombol JEV boleh asinkron, bukan kewajiban job infrastructure.

**Detail terbuka:** execution model, serialization/canonical hash, invalidation saat alias `jev-latest` berubah, lifetime cache dan policy hasil parsial belum final; set compare melalui Q27.

**Saran:** factual context dan skor deterministik tersedia independen dari JEV. Gunakan precompute/cache default snapshot bila memungkinkan; tanggal lain dapat on-demand dengan status pending yang jelas. Pilih mekanisme sync/async paling sederhana yang memenuhi latensi nyata setelah POC, bukan langsung membuat job infrastructure baru.

Terapkan kunci final PRD: hash atas bundle input aktual, versi pertanyaan, model dan snapshot. Usulan detail canonicalization: bundle mengikat deal, evidence/precedent IDs beserta isi/revisi, data version dan scope; versi pertanyaan mengikat rubric/prompt; snapshot mengikat `as_of` dan dataset publish version. Simpan alias dan resolved model version bila tersedia, tanpa mengarang metadata. Request set compare/urutan Choice masuk hash Q27. Tambahkan access scope bila retrieval berbeda per akses. Deduplicate request identik, batasi timeout/retry dan cegah request lama menimpa snapshot baru.

Saat model gagal, pertahankan komponen deterministik valid; hasil semantik `unavailable/not_evaluated` sesuai PRD, dengan error reason. Detail enum operasional `provider_error/invalid_output/pending` masih usulan Q18. `insufficient_evidence` berbeda dari kegagalan provider dan bukan label readiness ordinal.

**Alasan:** UI temporal dapat memicu panggilan berulang; pemisahan status mencegah biaya berlebihan dan kesuksesan palsu.

**Risiko jika ditebak:** perubahan tanggal mencampur skor, biaya melonjak dari render ulang, dan outage JEV tampil sebagai readiness rendah.

**Dibutuhkan:** sebelum assessment frontend dan readiness health. Enum/status/polling final disahkan melalui Q18.

## 5. Akses, operasi dan bukti penerimaan

### Q21: Proteksi demo dan scope akses minimum bagaimana?

**Final v1.0:** multi-tenant production auth **di luar scope MVP**; aplikasi mendukung persona Sales/VP tanpa execution/approval workflow. Read-only tidak menghapus kebutuhan proteksi data, secret dan paid-model endpoints.

**Detail terbuka / pertanyaan:** demo lokal/private atau dibuka ke jaringan? Gate akses dan pembatasan model apa yang diperlukan? Jika ada login/scope nyata, bagaimana token/session, pemetaan pengguna ke employee, evidence lintas deal dan ownership Copilot ditegakkan server-side? Tidak ada provider auth yang final.

**Saran (usulan):** pilih proteksi paling kecil yang memadai untuk exposure aktual: demo private dengan gate akses dan rate/budget limit, atau token backend yang diverifikasi jika deployment memerlukan identitas pengguna. Jangan membangun multi-tenant production RBAC sebagai prasyarat P0. Managed auth hanya alternatif bila login benar-benar diperlukan; Supabase Auth tidak otomatis mengikuti pilihan storage.

Persona switcher bukan otorisasi. Role/scope nyata tidak berasal dari body/localStorage. Jangan membuka paid-model proxy anonim atau menjadikan employee dataset akun login otomatis. Bila history disimpan, periksa ownership/session pada setiap pengambilan dan isolasi antaruser; session ID saja bukan izin. Retention Q22 dan snapshot policy Q17 tetap berlaku.

**Alasan:** kontrol demo proporsional menjaga scope dua orang sambil mengurangi kebocoran dan penyalahgunaan biaya.

**Risiko jika ditebak:** demo gate diklaim auth produksi, role dapat dipalsukan, history/evidence lintas pengguna bocor, atau biaya model dipakai publik tanpa batas.

**Dibutuhkan:** desain fase 0, selesai sebelum deployment terekspos pada fase 7. **Owner usulan:** O2, review joint; mekanisme/vendor tetap terbuka.

### Q22: Bagaimana biaya model, privacy, sanitasi, secret, dan log dibatasi?

**Pertanyaan:** Data apa boleh dikirim ke provider? Apakah full thread/nama/email perlu dikirim? Budget/rate limit berapa? Apa retention prompt/response/log? Bagaimana secret disediakan dan error vendor dibersihkan?

**Final v1.0:** `TYPESAFE_API_KEY` hanya backend/env, tidak di Git/browser; JEV menerima bundle bukti terbatas, bukan seluruh data mentah. Fixture buatan harus terisolasi dan tidak diklaim data kompetisi.

**Detail terbuka:** budget/limit, data-sharing/retention provider, sanitasi dan logging. Dataset kompetisi yang dideskripsikan sintetis tetap tidak membuat credential atau panggilan API bebas risiko/biaya.

**Saran:** kirim excerpt/metadata minimum yang mendukung tugas, bukan seluruh dataset. Secret lewat environment/config deployment, tidak di Git/frontend. Tetapkan limit request/body/context size, timeout, concurrent model calls, quota per user/global, dan cache sebelum exposure publik. Angka batas disetujui berdasarkan provider/rate limit/anggaran tim, bukan dikarang di dokumen.

Log request ID, duration, error class dan versi yang diperlukan audit; jangan default log authorization header atau full prompt/raw PII. Response yang disimpan untuk audit mengikuti kebijakan akses/retention. Source text dianggap untrusted; sanitasi HTML dan jangan ikuti instruksi yang tersisip pada email.

**Alasan:** endpoint read-only pun dapat menimbulkan tagihan dan data exposure. Synthetic dataset bukan alasan membocorkan credential.

**Risiko jika ditebak:** model proxy abuse, credential masuk log, data terkirim terlalu banyak, dan API keys terekspos lewat health/error.

**Dibutuhkan:** sebelum panggilan eksternal skala besar dan deployment publik. Catat implikasi external provider/data sharing dalam runbook.

### Q23: Bagaimana gold cases, test environment, dan hasil acceptance dibuktikan?

**Pertanyaan:** Siapa memberi label evidence/readiness/action dan memeriksa expected claims secara independen? Apa regression baseline dan environment nyata untuk unit/integration/E2E/live JEV? Bagaimana fixture 10%→14% terisolasi dan hasil acceptance direkam tanpa mengklaim mock sebagai live?

**Final v1.0:** unit/integration/E2E serta evaluation harness A/B/C termasuk P0. Data/JEV asli perlu bukti pengujian; jawaban model bukan ground truth. Fixture sintetis 10%→14% berlabel `DATA UJI`, terpisah dari P02 aktual.

**Detail terbuka:** labeler/reviewer, environment/command, test coverage teknis dan pelaksana UI/E2E/evaluasi belum ditetapkan. Ownership gap dirinci Q29; scaffold bukan bukti DoD.

**Saran:** siapkan kumpulan kasus kecil berlabel manusia, mencakup P02 approval scope, P04 reference gate/consent, P01 employment, P05 unknown, serta contradicting/ambiguous/future evidence. Tambahkan regresi: chat pada snapshot Oktober, mundur ke Agustus, lalu tanyakan follow-up; approval September tidak boleh terbawa lewat history. Tambahkan pergantian deal/user dan response model terlambat. Catat source IDs dan expected claim, bukan hanya kalimat jawaban yang di-hardcode. Simpan fixture sintetis pada test-only path dan jangan ingest ke dataset demo aktual.

Usulan split: O1 membuktikan join/temporal/idempotency/provenance dan expected facts; O2 membuktikan API, rule, provider validation/cache, serta live JEV pada environment yang tersedia. Joint meninjau label/rubrik dan test batas handoff; pelaksana UI/E2E ditetapkan lewat Q29, bukan mengasumsikan orang frontend ketiga. Tambahkan tes action dedup/cutoff, <2 opsi, Score/Choice berbeda, tie, partial failure, consent unknown dan cache set berubah. Catat command, tanggal aktual uji, versi dataset/model, pass/fail dan blocker; mock tetap terpisah dari live.

**Alasan:** acceptance MVP membutuhkan bukti sistem bekerja, bukan keberadaan adapter atau demo yang telah dihafal. Output model dapat berubah, sehingga penilaian harus memeriksa dukungan bukti dan allowed labels.

**Risiko jika ditebak:** fixture diklaim sebagai fakta perusahaan, regresi temporal tidak terdeteksi, atau API model dianggap sukses dari unit mock.

**Dibutuhkan:** sebelum sign-off; gold cases mulai sejak desain importer/assessment. Tidak perlu menentukan target win-rate atau causal uplift baru.

### Q24: Bagaimana dataset, config, import publish dan readiness disediakan?

**Pertanyaan:** `READMEHack.md` ada di mana? Bagaimana tim mendapatkan dataset yang di-ignore Git? Directory input/config storage apa? Bagaimana import failure, publish versi, migrasi, backup/reset demo, dan `/healthz` vs `/api/health` ditangani?

**Final v1.0:** inventaris file aktual dilakukan sebelum ingest, jumlah/checksum/FK/tanggal/error dicatat, import idempotent dan snapshot maksimum `2026-10-01`. PRD menyebut `DATA_DIR`, `SNAPSHOT_DATE`, URL graph, model dan `TYPESAFE_API_KEY`; ini kebutuhan konfigurasi, bukan bukti env/wiring sudah diimplementasikan. Docker Compose hanya disertakan jika digunakan/teruji.

**Detail terbuka:** distribusi data yang tidak ikut clone, lokasi input, publish atomik, reset/demo dan readiness. Audit dokumen sebelumnya menyebut kamus `HACKATHON PENS 2026/dataset_kasirnusa/README.md` sebagai kandidat pengganti `READMEHack.md`; kesetaraan/kelengkapan perlu dikonfirmasi pada inventaris aktual. Health scaffold statis bukan bukti data/graph/JEV siap.

**Saran:** konfirmasi apakah kamus tersedia memang pengganti referensi PRD. Sediakan onboarding/runbook untuk mendapatkan dataset melalui jalur terkontrol, lokasi input configurable, manifest/checksum versi, serta perintah import/rebuild yang repeatable. Jangan mengubah semua dokumen/dataset menjadi tracked otomatis hanya agar deployment berhasil.

Pisahkan import staging dari active dataset; publish hanya setelah integritas lulus sehingga request tidak membaca separuh import. Cara atomic publish disesuaikan engine, tidak perlu otomatis menambah message broker. Buat readiness membaca keadaan dataset/graph dan status konfigurasi/integrasi model yang benar, tanpa secret atau paid call pada setiap probe. Pertahankan liveness ringan; putuskan compatibility route `/healthz` dan status HTTP degraded pada kontrak API.

**Alasan:** kode benar tetap tidak dapat demo tanpa file sumber dan config; import parsial dapat menghasilkan kesimpulan bisnis salah.

**Risiko jika ditebak:** clone repo dianggap siap padahal dataset tidak ada, health palsu `ok`, dataset setengah terbit, dan demo tak dapat dibangun ulang.

**Dibutuhkan:** sebelum import reproducible/onboarding fase 0–2 dan deployment fase 7. **Owner usulan:** joint; O1 manifest/publish/data quality, O2 env/readiness/runtime. Env PRD selain `ADDR` scaffold dan command ingest masih perlu wiring/validasi; jangan mengklaim telah tersedia.

## 6. Action comparison dan handoff dua orang

### Q25: Bagaimana ActionTemplate/ActionOccurrence, kategori dan dedup dimodelkan?

**Final v1.0:** setiap ActionTemplate mempunyai sedikitnya satu ActionOccurrence berbukti dari tindakan/usulan nyata pada sumber. Occurrence memuat tindakan, aktor, target, waktu, outcome observed nullable dan evidence IDs. Status historis `requested/offered/approved/rejected/applied/completed` dipertahankan; outcome tidak diketahui bukan sukses. Kategori katalog hanya muncul jika ada preseden aktual.

**Detail terbuka / pertanyaan:** apakah template node graph atau record katalog yang menunjuk occurrence? Bagaimana stable `action_id`/`occurrence_id`, taxonomy/version, granularity template, link ke Event/Decision, revisi dan dedup lintas sumber? Apa yang membedakan dua catatan satu tindakan dari dua tindakan berulang yang memang berbeda?

**Saran (usulan):** O1 menyusun registry template–occurrence–evidence yang dapat ditraverse O2 lewat ID stabil Q07/Q30. Pisahkan kategori action dari `event_type`; contoh `REFERENCE_REQUEST`/`PILOT_OFFER` bukan bukti kategori itu ada di dataset. Template mengelompokkan pendekatan tanpa membuang aktor, target, nilai, scope dan status masing-masing occurrence. Dedup fakta yang sama berdasarkan source linkage/claim identity, bukan hanya teks mirip; occurrence berulang pada waktu/target berbeda tidak digabung. Simpan extraction/taxonomy version dan ambiguity; representasi fisik disepakati Q09. Jangan membuat template tanpa occurrence demi melengkapi menu.

**Alasan:** katalog yang stabil memisahkan pendekatan reusable dari kejadian historis, sehingga preseden, highlight graph dan cache dapat ditelusuri.

**Risiko jika ditebak:** satu decision dihitung dua kali, tindakan berbeda dilebur, opsi buatan dianggap sejarah, atau status approved diwariskan ke semua occurrence/template.

**Dibutuhkan:** saat ekstraksi fase 1, sebelum katalog/API compare fase 4. **Owner usulan:** O1 model/extraction/dedup; O2 meninjau kontrak consumer.

### Q26: Bagaimana relevance/eligibility candidates dan precedent cutoff ditentukan?

**Final v1.0:** candidates berasal dari katalog historis yang relevan dengan blocker, jenis pembeli, tahap atau bukti semantik; preseden/aktor/tanggal/evidence ditampilkan. Preseden akun lain berlabel `analog_precedent`, bukan fakta/approval deal ini. Jika <2 pilihan sah, tampilkan 0/1 beserta alasan; jangan menciptakan opsi untuk memenuhi compare.

**Detail terbuka / pertanyaan:** apa kriteria relevance minimum, batas jumlah candidates/precedents, dan eligibility untuk **dibandingkan**, bukan dieksekusi? Apakah setiap occurrence, evidence dan outcome pendukung terikat cutoff yang sama? Bagaimana stage historis unknown, konflik, account-only evidence, atau consent unknown memengaruhi kandidat?

**Saran (usulan):** bedakan (1) dukungan preseden, (2) alasan relevance, (3) eligibility compare dan (4) feasibility/policy Q28. Mulai dari rule yang dapat dijelaskan atas blocker/target/konteks yang known; jangan menjadikan stage snapshot-only syarat historis tanpa provenance. O1 mengembalikan candidate/precedent IDs, match reasons, unknowns dan batas cutoff; O2 memvalidasi lagi pilihan saat compare. Batasi seluruh bukti deal, occurrence, status/outcome dan penggunaan analog terhadap `as_of`/snapshot versi yang sama; outcome yang baru diketahui kemudian tidak dibawa mundur. Template global hanya menjadi kandidat historical bila memiliki occurrence berbukti yang valid pada cutoff. Semantic rerank/batas numerik perlu evaluasi, bukan threshold yang dikarang.

**Alasan:** relevance yang dapat ditelusuri menjaga pilihan masuk akal tanpa mencampur preseden, izin dan kejadian masa depan.

**Risiko jika ditebak:** katalog Oktober memberi solusi/outcome yang belum ada pada snapshot Agustus, analog dianggap approval saat ini, atau P05 memperoleh kandidat tanpa bukti relevan.

**Dibutuhkan:** sebelum action-candidates dan compare fase 4. **Owner usulan:** joint; O1 retrieval/cutoff/match, O2 validasi request dan policy exposure.

### Q27: Bagaimana rank Score vs Choice, ties, hasil parsial dan cache set 2–4?

**Final v1.0:** pengguna memilih 2–4 opsi historis sah. JEV Score ordinal 0..4 per opsi memakai rubrik sama dan `suitability_score_100 = round(25 * jev_score)`; bukan softmax antaropsi. Choice terpisah atas set terpilih memberi preferensi relatif, bergantung pada alternatif. Jika urutannya berbeda dari Score, tampilkan keduanya dan tandai perlu telaah, bukan buat konsensus. <2 opsi sah tidak menghasilkan perbandingan palsu.

**Detail terbuka / pertanyaan:** rank utama berbasis raw Score atau angka rounded, bagaimana tie/equal rank dan stabilitas urutan? Bagaimana duplicate/unknown action ID, 1 atau >4 pilihan, Score gagal pada satu item, Choice gagal/invalid dan retry parsial? Bagaimana set serta urutan opsi masuk cache agar distribusi Choice tidak dipakai pada set berbeda?

**Saran (usulan):** rank utama berdasarkan raw suitability Score menurun; tie disajikan sebagai seri dengan stable `action_id` untuk urutan tampilan, bukan diklaim satu opsi lebih baik. Choice tampil terpisah dan tidak menjadi tie-break tersembunyi. Rank hanya hasil valid; bila satu Score gagal, pertahankan item sukses dengan penanda partial dan jelaskan scope rank, bukan beri nol pada item gagal. Jika <2 Score valid, jangan menyatakan ranking lengkap/sukses. Choice yang valid tetap terkait **set request asli**, tidak diam-diam ditafsir ulang sebagai distribusi subset sukses. Failure Choice tidak mengubah Score; reason/status serta UX parsial disepakati Q18.

Validasi server terhadap 2–4 **ID unik**, existence, snapshot dan eligibility Q26; usulan tolak duplicate/invalid tanpa silent truncation. Cache v1 Q20 menghash bundle deal + bukti masing-masing action, versi pertanyaan/rubrik, model dan snapshot. Usulan canonicalize urutan IDs/bundle sebelum membangun payload Choice, lalu hash payload/set yang benar-benar dinilai; jika urutan dipertahankan, urutan itu ikut hash. Cache Score per opsi boleh reusable hanya untuk input identik; cache Choice selalu milik set 2–4 tertentu. Jangan menormalisasi ulang preferensi cached setelah opsi ditambah/dihapus.

**Alasan:** pemisahan Score/Choice dan cache per set menjaga perbedaan nilai rubrik per opsi berdasarkan bukti vs preferensi relatif antaralternatif.

**Risiko jika ditebak:** rank berganti karena rounding atau urutan input, failure menjadi low score, distribusi set dua opsi muncul pada set empat, atau perbedaan Score dianggap uplift closing.

**Dibutuhkan:** sebelum compare runtime fase 4 dan UI fase 5. **Owner usulan:** O2 ranking/cache/status; joint meninjau tie/partial semantics dan pelaksana UI Q29 mengonfirmasi UX.

### Q28: Bagaimana policy feasibility/consent dan requires_validation berlaku untuk compare?

**Final v1.0:** >10% membutuhkan approval VP Sales yang terkait; calon reference tidak berarti consent. Tanpa bukti izin → `consent: unknown`. Preseden akun lain bukan approval. Kandidat dapat berlabel `requires_validation` bila bukti/kelayakan tidak lengkap, tidak ditampilkan seolah siap dieksekusi. Ranking membantu keputusan manusia, **bukan execution**.

**Detail terbuka / pertanyaan:** flag/prasyarat apa yang machine-checkable, bagaimana consent scope/waktu dan approval validity dibuktikan, dan kapan opsi disembunyikan vs tetap bisa dibandingkan dengan warning? Bagaimana membedakan bukti preseden tidak ada dari preseden ada tetapi feasibility belum terverifikasi?

**Saran (usulan):** baseline compare memerlukan occurrence/evidence valid Q25–Q26; `requires_validation` tidak menggantikan preseden yang hilang. Pisahkan status dukungan sejarah, relevansi, policy dan feasibility. O1 memasok fakta approval/consent beserta scope/provenance, O2 menerapkan warning/prasyarat. Opsi berbukti dengan consent unknown atau approval belum ada dapat tetap **dipertimbangkan untuk compare** dengan label/prasyarat eksplisit bila policy compare disepakati; jangan menyebutnya executable/authorized. Tampilkan flags, alasan, evidence dan unknowns per item, termasuk dalam penjelasan Score/Choice. Approval 10% tidak berlaku untuk request 14%; consent pelanggan pembanding tidak diwariskan ke referensi lain. Jika ada bukti larangan yang relevan, nyatakan kendala itu, bukan menyembunyikannya di angka suitability.

**Alasan:** suitability semantik dan izin operasional berbeda; warning eksplisit memungkinkan membandingkan pendekatan tanpa mengarang otorisasi atau SOP.

**Risiko jika ditebak:** skor tinggi dianggap boleh menghubungi pelanggan/mengirim diskon, consent palsu, atau `requires_validation` dipakai untuk meloloskan opsi tanpa sejarah.

**Dibutuhkan:** sebelum eligibility/policy payload fase 4 dan penjelasan UI fase 5. **Owner usulan:** joint; O1 fakta/scope historis, O2 validator/flags, review kasus kebijakan Q23.

### Q29: Siapa memiliki UI/E2E dan evaluasi A/B/C di rencana dua orang?

**Final v1.0:** UI temporal interaktif, evidence/score drawer, action comparison dan Copilot tetap P0. E2E menguji alur panel dua arah dan perubahan snapshot atomik. Evaluasi memakai **A unguided** (CRM/CSV/ringkasan tanpa graph/JEV), **B graph-guided** (graph/timeline/evidence tanpa JEV/compare JEV), **C AI-guided** (Relio lengkap). Pembagian user baru mencakup O1 graph/data dan O2 backend, belum pelaksana UI/evaluasi.

**Detail terbuka / pertanyaan:** siapa mengimplementasikan UI, mengonsumsi DTO dan menjalankan E2E; apakah scope salah satu orang diperluas atau ada bantuan terpisah yang disetujui? Siapa menyusun tugas/kunci/rubrik independen, mode A/B/C dan harness, merekrut peserta jika memungkinkan, serta meninjau laporan? Bagaimana backend mencegah skor JEV bocor ke kondisi B?

**Saran (usulan):** joint menetapkan pelaksana serta reviewer eksplisit sebelum handoff API; jangan otomatis menugaskan UI kepada O2 atau mengasumsikan orang ketiga. O1 dapat menyiapkan expected facts/evidence paths dan data pembanding; O2 dapat menyiapkan kontrak/mode API, logging metrik dan harness. Itu kontribusi yang disarankan, bukan pengganti owner UI/E2E. Kunci faktual dibuat manual dari sumber dan direview silang/independen, bukan output JEV. Pilih 6–10 kasus setara sesuai PRD; gunakan counterbalanced order bila peserta mencoba beberapa kondisi. Pastikan kondisi B benar-benar menonaktifkan output JEV, bukan sekadar menyembunyikan judul.

Siapkan kit P0 `evaluation/tasks.md`, `evaluation/answer_key.md`, `evaluation/score_rubric.md`, `evaluation/results_template.csv`, `scripts/evaluate_results.*` dan `evaluation/limitations.md`. Ukur evidence retrieval, kualitas keputusan menurut rubrik independen, waktu bila benar-benar tercatat, policy compliance, uncertainty handling serta usefulness/calibration. Pengumpulan peserta nyata bergantung ketersediaan; laporkan jumlah/desain aktual dan keterbatasan, jangan mengarang hasil, durasi pekerjaan atau win-rate uplift.

**Alasan:** dua pekerjaan backend tidak otomatis menghasilkan produk UI end-to-end atau evaluasi independen; ownership eksplisit menutup gap P0.

**Risiko jika ditebak:** API selesai tetapi demo/DoD gagal, eksperimen A/B/C tercampur, hasil model dijadikan kunci, atau tugas UI jatuh ke orang yang tidak menyetujuinya.

**Dibutuhkan:** sepakati owner sebelum kontrak UI dikonsumsi pada fase 0–2; bukti UI/E2E fase 5 dan kit evaluasi fase 6–7. **Owner usulan:** joint untuk keputusan; pelaksana UI/E2E/evaluasi **belum ditetapkan** dan tetap blocker kelengkapan P0.

### Q30: Apa contract handoff versi/data/provider interface dan ownership split?

**Final v1.0:** semua owner menyerahkan endpoint/schema/contoh data teruji; ID/provenance stabil, snapshot konsisten, ETL repeatable dan JevAdapter server-side memakai evidence dari graph. Pembagian user ialah O1 context graph/data dan O2 aplikasi backend, bukan dua implementasi domain yang berbeda.

**Detail terbuka / pertanyaan:** artefak versi mana menjadi sumber bersama; boundary O1→O2 lewat graph repository/provider interface atau hasil publish apa? Siapa memiliki schema/temporal query, event/action extraction, coverage/denominator inputs, policy facts, retrieval dan API serialization? Bagaimana kedua orang menyepakati kompatibilitas/revision, kesiapan data dan error tanpa import parsial atau vendor API bocor ke DTO?

**Saran (usulan):** buat kontrak kecil sebelum implementasi silang, bukan abstraction platform baru. O1 memiliki ETL/entity resolution, graph schema/query temporal, source lookup, action occurrence/catalog/relevance dan manifest publish. O2 memiliki HTTP DTO/error/status, rubric calculation/orchestration, JEV/Copilot provider adapter, ranking/cache/policy presentation dan proteksi runtime. Joint menyetujui semantik snapshot, kriteria known, eligibility/consent/approval, perubahan kontrak dan acceptance; keputusan UI tetap Q29. Query storage berada pada boundary graph yang dibuktikan O1 dan dikonsumsi O2, tidak diduplikasi di controller.

Handoff yang disarankan memuat:

- Schema/contract version, stable node/edge/event/action/evidence IDs dan aturan null/unknown/ambiguous.
- Dataset/publish version, checksums, snapshot/cutoff, availability field, laporan kualitas dan indikator partial/ready.
- Bentuk keluaran provider graph untuk deal context, timeline, evidence, candidates/precedents dan retrieval; error/not-found/empty/truncated dibedakan. Signature aktual disepakati, bukan nama method SDK rekaan.
- Bundle masukan JEV/Copilot dengan scope, evidence/precedent IDs, kutipan/status/waktu; O2 memvalidasi sebelum panggilan. Provider interface aplikasi terpisah dari kontrak HTTP vendor dan DTO frontend.
- Versi taxonomy/extraction, rubric/pertanyaan/model yang relevan untuk cache Q20/Q27, contoh data teruji berlabel real vs fixture, dan test kontrak yang dijalankan O1+O2.

Mulai dengan satu vertical slice bersumber: deal → snapshot graph/timeline/evidence → kandidat berbukti → input assessment. Pilih fixture terlabel bila sumber belum tersedia, tanpa klaim integrasi data/JEV asli. Sepakati kebijakan perubahan/breaking version dan jangan publish dataset sampai pemeriksaan integritas lulus Q24.

**Alasan:** satu boundary versi/provenance memungkinkan paralelisme dua orang tanpa dua definisi snapshot atau kebutuhan menunggu seluruh implementasi selesai.

**Risiko jika ditebak:** ID/schema berbeda, API membaca separuh import, kedua orang memiliki cache/rule berbeda, atau perubahan ETL diam-diam merusak citations dan ranking.

**Dibutuhkan:** sebelum handoff fase 0–2, diperbarui sebelum assessment/compare/Copilot fase 3–6. **Owner usulan:** joint; O1 producer data/graph, O2 consumer API/provider.

## 7. Mana yang perlu dijawab lebih dahulu?

### Kelompok keputusan berdasarkan dependency, bukan kalender

1. **Data/storage/handoff:** Q01–Q02, Q07, Q09, Q24–Q25, Q30 — usulan ditangani sebelum schema/import dikunci karena risiko dua pipeline atau contract drift.
2. **Waktu/identitas/history:** Q03–Q06, Q08, Q11 — usulan diuji sebelum query/snapshot dipakai agar current facts/outcome tidak bocor ke sejarah.
3. **Rubrik/validasi/policy:** Q10, Q12–Q16, Q26–Q28 — usulan diselesaikan sebelum assessment/compare ditampilkan agar angka final diterapkan tanpa approval atau label palsu.
4. **API/runtime/UI/evaluasi:** Q18–Q20, Q23, Q29 — usulan owner dan DTO ditetapkan sebelum consumer/UI bergantung padanya karena backend saja tidak memenuhi P0.
5. **Provider/operasi:** Q15, Q17, Q21–Q22, Q24 — usulan POC akses/data dan proteksi dimulai paralel pada fase audit karena blocker eksternal tidak diselesaikan hanya dengan kode.

### Blocker yang harus tetap terlihat

- **JEV live belum terverifikasi:** endpoint/payload PRD dan mock bukan bukti keberhasilan. POC authenticated + validasi respons menjadi blocker sign-off; credential/akses/limit yang belum tersedia perlu dicatat Q15/Q22.
- **Data/publish aktual:** bila sumber atau manifest tidak tersedia, klaim preseden, identitas dan P01–P05 belum bisa diterima. Audit/akses/reimport nyata Q24/Q30 adalah bukti yang dibutuhkan, bukan data buatan.
- **Kurang dari dua opsi sah pada deal/cutoff:** blocker compare pada deal tersebut, bukan alasan menciptakan katalog palsu; tampilkan 0/1 dan alasan Q25–Q28.
- **Owner UI/E2E/evaluasi belum ditetapkan:** rencana O1/O2 belum menutup seluruh P0. Keputusan Q29 harus eksplisit sebelum menyebut MVP end-to-end selesai.
- **Exposure belum aman:** proteksi demo/secret/budget Q19/Q21/Q22 wajib sebelum akses jaringan/provider berbayar dibuka; bukan tuntutan auth multi-tenant produksi.

### Yang dapat dilanjutkan tanpa membuka ulang keputusan final

Usulan berikut aman pada fase audit/data karena mengikuti PRD; risikonya adalah pekerjaan dianggap bukti live padahal belum tervalidasi, sehingga artefak/test harus berlabel jelas:

- Inventaris/validasi CSV/JSONL, FK, checksum dan pemetaan account→deal aktual oleh O1.
- Parser/normalizer teruji, provenance, ambiguity dan idempotency; framework/runtime mengikuti keputusan Q01/Q02, bukan otomatis Go importer.
- Model nullable, lima dimensi coverage dan lima level readiness final; kriteria known/cutoff/history disepakati Q11–Q14.
- Test kebijakan >10%, fixture 10%→14% terisolasi, nilai/bobot urgency final dan unknown vs zero oleh joint.
- Kontrak minimal graph→backend Q30 dan adapter siap diuji oleh O2; POC JEV nyata segera setelah akses tersedia, bukan mock yang disebut live.

### Yang jangan dilakukan sebelum keputusan/bukti terkait

- Migration engine tertentu sebelum Q01/Q09 atau migrasi stdlib→Gin tanpa justifikasi Q02.
- Mengisi field/history/precedent lampau dari snapshot sekarang sebelum Q03–Q06/Q26.
- Mengubah denominator prospek aktif, rentang 7 hari, lima dims/level, atau formula final tanpa change request.
- Membuat action tanpa occurrence berbukti, mewariskan approval/consent atau membangun execution/SOP di luar PRD v1.0.
- Mengarang response/method JEV, hasil POC, metrik latency, jadwal, peserta evaluasi atau win-rate uplift.
- Mengekspos evidence/model API sebelum kontrol demo/data/biaya Q19/Q21/Q22, atau menyatakan UI/E2E selesai hanya karena API tersedia.

## 8. Template mencatat keputusan

Saat detail teknis disepakati, catat bukti dan batasannya; owner dokumen terkait dapat memperbarui [arsitektur](architecture.md), [rencana dua orang](development-plan.md) dan kontrak terkait. Tidak perlu membuka kembali keputusan final PRD tanpa change request.

```text
ID pertanyaan:
Status: open / partially clarified / disetujui / ditunda / perlu spike
Keputusan final PRD yang tetap berlaku:
Detail terbuka dan usulan yang diputuskan:
Alasan dan alternatif yang ditolak:
Batasan/risiko yang diterima:
Owner O1 / O2 / joint dan pemilik persetujuan:
Tahap/dependency saat dibutuhkan:
Tanggal keputusan aktual (bukan estimasi):
Dampak: schema / API / provider / prompt / cache / UI / tests / evaluasi / deployment
Versi data/kontrak/model atau artefak yang diperbarui:
Bukti validasi aktual atau blocker tersisa:
```

Dokumen ini memuat **30 pertanyaan (Q01–Q30)**, bukan permintaan agar user menjawab semuanya sekaligus. Mulai dari dependency yang memblokir pekerjaan berikutnya; saran dan owner tetap usulan sampai disepakati, integrasi tetap belum verified sampai ada hasil uji nyata.
