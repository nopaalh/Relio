# Relio — Draft soal evaluasi faktual O1

**DRAFT — menunggu review Orang 2 / reviewer evaluasi**  
Tanggal: 9 Oktober 2026. Baseline file kompetisi: audit yang diverifikasi sebelumnya; source belum dapat dibaca ulang pada turn ini. [Status dan sumber](README.md).

Delapan tugas ini adalah bahan evaluasi berbasis sumber untuk rencana kondisi A unguided, B graph-guided, dan C graph+JEV-guided. Mode/harness/UI kondisi tersebut belum diimplementasikan oleh dokumen ini. Soal dan tanggal sama lintas kondisi; akses sumber harus setara sesuai scope yang nanti disepakati. Jangan menunjukkan kunci jawaban kepada peserta.

Untuk setiap tugas, minta peserta mengisi jawaban singkat, source record/field yang mendukung, serta informasi yang belum diketahui. Pengukuran waktu/hasil hanya diisi jika benar-benar direkam. Belum ada peserta, durasi, skor evaluasi, atau klaim uplift.

| Task | Cutoff | Pertanyaan faktual |
|---|---|---|
| EVAL-01 | 2026-09-01 | Pada akun Nirwana (P04 / DL-004), bukti interaksi apa yang tersedia sampai tanggal ini? Apakah dataset pada cutoff ini sudah mendukung klaim bahwa pengadaan ditunda sampai ada referensi? |
| EVAL-02 | 2026-09-22 | Apa syarat yang disampaikan pengirim Nirwana sebelum tanda tangan? Apakah sumber itu juga membuktikan pelanggan referensi sudah consent atau referensi sudah diberikan? |
| EVAL-03 | 2026-10-01 | Apa yang bisa dan belum bisa disimpulkan tentang PT Distribusi Sumber Rejeki (P05 / DL-005) ketika daftar account interactions kosong? |
| EVAL-04 | 2026-09-28 | Email internal Teras Kafe (P02 / DL-002) menyebut diskon 20%. Apakah email itu permintaan atau persetujuan? Bukti tambahan apa yang diperlukan untuk menyatakan approved? |
| EVAL-05 | 2026-10-01 | CRM akun C01 menunjuk K017 sebagai champion, sementara current contact berada pada P01. Bagaimana menjelaskan perbedaan ini tanpa memperbaiki atau menebak data sumber? |
| EVAL-06 | 2025-08-12 | Keputusan D-2025-06 pada C23/DL-007 mencatat opsi Starter/pilot. Apakah teks “Deal menang Sep 2025” boleh dipakai sebagai outcome yang sudah terjadi pada cutoff ini? |
| EVAL-07 | 2026-10-01 | Nilai transaksi offline tersinkron kosong pada outlet offline-disabled. Apakah blank tersebut berarti ada nol transaksi offline yang teramati? Mengapa? |
| EVAL-08 | 2025-12-01 | Apa perbedaan bukti keputusan D-2025-11/I0061 dan kontrak K-C01? Apakah tanggal approval dan awal kontrak membuktikan application terjadi pada tanggal yang sama? |

Task IDs adalah ID evaluasi teknis. Native source IDs hanya referensi file kompetisi, bukan graph/evidence IDs yang sudah disetujui.

## Instruksi reviewer

Periksa [draft kunci jawaban](evaluation-answer-key-draft.md) terhadap source asli dan hashes pada source-cases JSON sebelum membekukan tugas. Review terutama snapshot-only facts, retrospective text, distinction request/approval/application, dan absence evidence. Tidak ada jawaban yang diturunkan dari output JEV.

Orang 2/reviewer menentukan rubrik/HTTP/UI condition controls; kunci fakta bukan rubrik “setuju dengan model”. Kasus consent unknown tidak boleh otomatis diberi jawaban false/denied. Task tentang record lama tidak memberi izin membuka seluruh row snapshot apabila kutipannya membocorkan future outcome.
