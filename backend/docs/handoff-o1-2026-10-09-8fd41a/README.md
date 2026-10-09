# Relio — Paket kasus sumber untuk handoff O1/O2

**DRAFT — menunggu persetujuan Orang 2**  
**Tanggal:** 9 Oktober 2026.  
**Scope:** dataset FIX, database bisnis statis disiapkan sekali; tanpa pipeline ETL.

Paket ini menyiapkan 15 kasus review/acceptance dari sumber yang diperiksa pada audit sebelumnya. Tidak membuat source Go, schema database, adapter, migration, API response fixtures, ataupun persetujuan kontrak baru. Folder handoff terpisah dipakai agar dokumen/source bersama tidak ditimpa ketika keadaan repo belum dapat dibaca ulang.

## Status bukti

- Baseline: [audit sumber](../dataset-audit.md), [JSON audit](../source-audit-snapshot.json), [mapping sumber](../source-mapping.md), dan [draft kontrak](../data-repository-contract-draft.md).
- File input diverifikasi pada audit sebelumnya dalam percakapan ini. Terminal pada turn ini gagal dengan `helper_unknown_error: setup refresh had errors`; source dan Git status belum dapat diperiksa ulang.
- Payload kasus disusun dan diperiksa dalam memori: case IDs unik, JSON round-trip, sumber/counterexample ada, P04 date sets sesuai cached audit, P05 tetap empty, serta missing offline counts konsisten.
- Ini bukan hasil test adapter, HTTP, database, atau JEV/Copilot. Periksa checksum source sebelum menjadikannya expected result integration tests.
- Hanya assertions source yang langsung reproduktif diperlakukan sebagai observasi. Proposed contract expectations tetap usulan; contoh label/status bukan DTO atau signature final.

## Isi paket

[Source cases JSON](source-cases-draft.json) memuat input hashes 15 file, source locators, expected observations, proposed expectations, dan forbidden interpretations. Case IDs `SRC-…` adalah ID test, bukan source/business/event IDs.

| Case | Fokus | Yang diuji/direview |
|---|---|---|
| SRC-01 | Deal/account | DL-004 berbeda dari P04. |
| SRC-02 | Temporal P04 | Tujuh date-only cutoffs; I0335 baru visible sejak 2026-09-22. |
| SRC-03 | Account-only linkage | Interaction tidak punya native deal_id; reply links tidak dibuat dari subjek. |
| SRC-04 | Reference/consent | Permintaan referensi tidak memberikan consent atau bukti delivery. |
| SRC-05 | P05 unknown | Empty account interactions tidak berubah menjadi readiness nol. |
| SRC-06 | Snapshot values | Stage/owner/ACV/outlets tidak otomatis historis. |
| SRC-07 | Champion stale | K017 pointer C01 terpisah dari current account P01. |
| SRC-08 | Email historis | Satu alamat unmatched muncul enam kali; identity alias tidak ditebak. |
| SRC-09 | Employment overlaps | Sepuluh overlap pairs dipertahankan sebagai sumber, bukan auto-repair. |
| SRC-10 | Missing numeric | Blank offline values bukan observed zero. |
| SRC-11 | Approval unknown | Missing explicit decision link bukan rejected/approved. |
| SRC-12 | Evidence future | Future quote tidak masuk model bundle historical. |
| SRC-13 | P02 request 20% | I0348 adalah request/proposal, approved value tetap unknown. |
| SRC-14 | Retrospective outcome | Winner text September tidak dipakai sebagai outcome pada Agustus. |
| SRC-15 | Approval/application | D-2025-11/I0061 terpisah dari K-C01 contract start. |

## Draft evaluasi faktual

Tambahan paket: [delapan soal peserta](evaluation-tasks-draft.md) dan [kunci faktual untuk reviewer](evaluation-answer-key-draft.md). Kunci bukan output model dan belum menjadi hasil studi peserta. Jangan memberikan file kunci kepada peserta ketika mengukur jawaban.

## Memakai paket saat Orang 2 tersedia

1. Review proposed expectations terhadap kontrak; source facts tidak menentukan HTTP statuses atau database engine.
2. Cek SHA256 semua source files terhadap `expected_input_hashes`. Jika berubah, tandai expectation stale dan audit ulang; jangan update expected outcome otomatis.
3. Reproduksi `source_observation` dari field/IDs native. Jangan menjadikan JSON handoff sebagai endpoint bisnis.
4. Setelah kontrak disepakati, O1 memetakan kasus ke adapter tests; O2 memetakan failure/unknown/context ke service/DTO tests.
5. Catat pass/fail/skip dari execution nyata. Test-only stub tidak membuktikan database/live JEV.

Command berikut contoh inspeksi read-only dari folder HeketonPens; belum dijalankan pada turn ini:

```powershell
$sourceInteractions = Get-Content -Encoding UTF8 -LiteralPath 'Datasets\interactions.jsonl' |
    ForEach-Object { $_ | ConvertFrom-Json }
$sourceInteractions |
    Where-Object { $_.account_id -eq 'P04' -and $_.tanggal -le '2026-09-01' } |
    Sort-Object tanggal,interaction_id |
    Select-Object interaction_id,tanggal,account_id
```

Expected source IDs: `I0284, I0314`. Mengganti cutoff ke `2026-09-22` menambahkan `I0335`. Ini source filter date-only, bukan final graph query, identity verification, atau access policy.

## Yang tetap menunggu keputusan

Nullable compatibility existing model, temporal boundary employment, account-only→deal rules, server access scope, taxonomy/relevance, query bounds dan engine/instance database tetap draft. Paket ini tidak menyetujui keputusan tersebut.

Tidak ada klaim comparison berhasil atau dua kandidat sah tersedia. Semua 15 kasus adalah bahan handoff/test specification; hasil runtime hanya boleh dilaporkan setelah benar-benar dijalankan.
