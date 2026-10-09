# Relio — Draft kunci faktual berbasis sumber

**DRAFT — menunggu review Orang 2 / reviewer evaluasi**  
Tanggal: 9 Oktober 2026. Ini reference answer berdasarkan source reads sebelumnya, bukan output JEV, observed user study, atau hasil query adapter. Terminal gagal dibuka pada turn ini; source files tidak diperiksa ulang. Hash baseline ada di [source cases](source-cases-draft.json); [soal peserta](evaluation-tasks-draft.md).

Kunci memisahkan field native yang terbaca dari semantik kontrak yang belum disetujui. Penilaian harus menerima jawaban equivalent dengan bukti benar dan uncertainty yang dijaga. Tidak menciptakan consent, approval, alias person verified, stage history, atau precise completion date.

## EVAL-01 — P04 sebelum pesan referensi

- Source IDs native account P04 hingga 2026-09-01: **I0284 (2026-08-10)** dan **I0314 (2026-08-28)**.
- **I0335 bertanggal 2026-09-22**; body penundaan referensi tidak tersedia pada cutoff task ini.
- Jawaban yang didukung: dua record tersebut tersedia; belum ada quote penundaan reference dari I0335 yang boleh dipakai pada cutoff ini. Ini tidak membuktikan penundaan atau permintaan referensi tidak pernah ada di dunia nyata.
- Locators: `interactions.jsonl[interaction_id=I0284/I0314/I0335].tanggal,account_id,isi`.
- Ketiganya account-scoped; native interaction `deal_id` tidak ada. Link ke DL-004 tetap aturan yang perlu disepakati.
- Hindari: menggunakan I0335 karena seluruh database statis sudah menyimpan record tersebut.

## EVAL-02 — Permintaan bukan consent atau delivery

- I0335 / 2026-09-22 / account P04 / `isi`:
  > Pak Bagus, Direktur Utama kami minta rekomendasi dari pengguna yang mirip dengan kami sebelum tanda tangan. Kami tunda dulu sampai ada referensi.
- Pengirim meminta rekomendasi pengguna mirip sebelum tanda tangan dan menyatakan menunda hingga ada referensi.
- Kutipan itu tidak menyebut pelanggan reference tertentu telah memberikan consent, reference sudah delivered, atau action sudah completed. Consent/delivery tetap unknown tanpa bukti tambahan yang scope/waktunya tepat.
- Kata “Direktur Utama kami” tidak otomatis mengidentifikasi K028 verified; matching jabatan adalah bahan linkage review.
- Hindari: suitability/reference request diperlakukan sebagai permission menghubungi pelanggan.

## EVAL-03 — Empty context P05

- Native CRM: `crm_deals[deal_id=DL-005].account_id=P05`, `dibuat=2026-09-26`, snapshot stage Lead, status Terbuka.
- Audit account filter pada interactions menemukan **0 record account P05**; explicit decision deal link juga kosong; daily usage akun P05 0 rows.
- Kesimpulan: ada opportunity CRM, tetapi bukti interaksi yang dapat dipakai dari account linkage tersebut belum memadai. Ketiadaan record tidak berarti Sales tidak pernah menghubungi atau prospect tidak serius.
- Stage/komersial adalah snapshot facts, bukan semua histori. Tidak memberi readiness level/score 0 untuk missing evidence.
- Hindari: menciptakan tindakan/percakapan tambahan untuk memenuhi demo/comparison.

## EVAL-04 — Request/proposal diskon 20%

- `interactions.jsonl[interaction_id=I0348]`: tanggal 2026-09-28, account P02, dari Citra ke alamat Andi, tipe email_internal.
- `isi`:
  > Pak Andi, untuk menutup Teras Kafe saya usul diskon 20% agar menyamai KasirPro. Mohon keputusan.
- Sumber adalah usul/permintaan keputusan, bukan jawaban persetujuan.
- Tidak ditemukan native `decision_log.deal_id=DL-002` pada audit. Itu absence link, bukan rejected, approved, atau bukti absolut tidak ada approval di sumber lain.
- Aturan >10% memerlukan VP approval berasal dari README/PRD. Untuk menyatakan approved diperlukan decision/evidence terpisah yang sesuai request, nilai, deal/scope dan waktu; preseden akun lain tidak mencukupi.
- Hindari: mengubah “mohon keputusan” menjadi approved atau menjalankan action komersial.

## EVAL-05 — Champion stale dan perpindahan kontak

- `crm_accounts[account_id=C01].champion_contact_id=K017`.
- `crm_contacts[contact_id=K017].account_id_saat_ini=P01`, current jabatan GM Operations.
- Native history K017: account C01 **2021-03-01 … 2026-08-15**; account P01 sejak **2026-09-01**, end kosong.
- Pointer CRM dan employment mempunyai makna/sumber berbeda. Catat mismatch/staleness sebagai limitation; jangan menganggap K017 masih active C01 contact/champion dari pointer saja.
- History tidak mempunyai native row ID; gunakan tuple source untuk inspeksi. End-day boundary tepat belum diputuskan.
- Email lama Kopi Lintas tidak menjadi alias verified otomatis hanya karena nama/local-part sama.

## EVAL-06 — Outcome masa depan dalam record lama

- `decision_log[decision_id=D-2025-06]`: tanggal 2025-08-12, account C23, deal DL-007, keputusan Disetujui, nilai “Paket Starter tanpa diskon, pilot 6 outlet”.
- `alasan` menyertakan “Deal menang Sep 2025.”; kalimat retrospective itu tidak membuktikan Won sudah terjadi pada 12 Agustus.
- Snapshot CRM DL-007 memiliki Closed Won / stage_since 2025-09-08; field ini bukan log transisi lengkap atau bukti tanggal outcome action kausal.
- Approval proposal tidak otomatis berarti pilot completed atau penyebab Won. Untuk historical evidence harus membatasi field/span/claim yang benar-benar diketahui pada cutoff sesuai kontrak.
- Hindari: quote seluruh alasan hanya karena record `tanggal` lebih awal; membuat precise outcome date dari bulan teks.

## EVAL-07 — Blank bukan observed zero

- README menjelaskan `product_usage_daily.transaksi_offline_tersinkron` kosong bila outlet tidak memakai mode offline.
- Source audit: **590 outlet mode tidak ×365 hari =215.350 blank values**. Ada 30 outlet mode ya; tidak ditemukan missing values pada rows kelompok itu dalam pemeriksaan sebelumnya.
- Blank berarti tidak tersedia/tidak applicable sesuai field context; bukan observed number 0. Jangan mengimputasi nol atau mengklaim transaksi count adalah omzet/laba.
- Locators: `outlets.outlet_id,mode_offline_aktif`; daily rows dengan account/outlet/date keys.
- Pemeriksaan ini hanya daily records pelanggan existing; tidak menghasilkan usage prospek.

## EVAL-08 — Approval dan application dipisahkan

- `decision_log[decision_id=D-2025-11]`: C01/DL-008, tanggal **2025-11-28**, diskon **15%**, explicit evidence link **I0061**.
- I0061 memuat persetujuan 15% dan komitmen roadmap; itu evidence keputusan terpisah.
- `contracts_billing[contract_id=K-C01]`: account C01, `decision_id=D-2025-11`, `diskon_pct=15`, `mulai=2025-12-15`.
- Pada task cutoff **2025-12-01**, tanggal awal kontrak masih sesudah cutoff. Jangan menyatakan application sudah terjadi pada tanggal approval atau memakai current contract snapshot sebagai past application tanpa temporal support.
- Native link/value match adalah bahan provenance. Ini tidak mengizinkan request lain 15% atau request P02/P04.
- Timestamp kapan setiap field pertama diketahui tidak tersedia; jangan mengarang recorded_at.

## Handoff dan limitations

Semua delapan tasks mempunyai source locators dan negative expectations. Source excerpts diturunkan dari pembacaan file kompetisi sebelumnya, tanpa mock business findings. Field availability dan response shape masih memerlukan kontrak; policy warning/scoring/cache/model orchestrations tetap Orang 2.

Reviewer memeriksa source hash/quotes, membekukan answer key dan menentukan rubric independen sebelum evaluasi peserta. Belum ada execution kondisi A/B/C, peserta, skor, waktu, P(win), effect size, atau win-rate uplift yang boleh dilaporkan.
