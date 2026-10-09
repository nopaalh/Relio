# Relio Frontend v1.0

Update 10 Oktober: English UI, VP Sales persona, tanggal di atas planet, date search/rail/slider, Context Explorer popup dan contextual Copilot drawer. Layout aktif desktop dua kolom timeline + graph; source evidence dibuka melalui popup. Ini menggantikan uraian tiga kolom dan Copilot tab bawah sebelumnya. Anti-slop project setup ada di [docs/anti-slop](../docs/anti-slop/README.md).

Next.js + TypeScript + React Flow. Mengikuti PRD Final MVP v1.0 dan GSM Universe v3.
Tagline: **Your deal universe. A clearer mind.** Logo Ribbon Flow R tetap resmi; tokens Universe v3 ada di ../Relio_GSM_Universe_v3/. Styling/motion/layout planet sepenuhnya FE dan tidak membutuhkan API BE tambahan.

## Jalankan lokal

Branch GitHub `frontend` mengunggah folder frontend saja. Dataset `dataset_kasirnusa`, dokumen GSM, dan direktori workspace lain tidak disertakan. Mode local membutuhkan dataset tersebut secara terpisah melalui DATA_DIR. Untuk backend yang sudah tersedia, gunakan RELIO_DATA_MODE=api dan RELIO_API_URL pada .env.local. Logo dan CSS brand yang dibutuhkan UI sudah berada dalam frontend.

Node.js >=22 disarankan (runtime yang diuji: 22.17.0).

```powershell
cd frontend
npm ci
npm run dev
```

Buka http://127.0.0.1:3000/deals. Detail contoh: http://127.0.0.1:3000/deals/DL-004.
Port yang terpakai bisa diberikan via npm run dev -- --port 3001.

Mode default local membaca ../dataset_kasirnusa dari server, bukan browser. Sumber asli sintetis; API JEV/Copilot belum terhubung, UI menampilkan unavailable dan tidak membuat skor atau jawaban palsu.

## Tersedia

- Ringkasan lima prospek, ACV faktual, daya tarik deterministic, coverage lima dimensi.
- Search/filter/sort, no match, retry, deep link dan pemulihan filter daftar.
- Detail deal, timeline type/actor/status, source quotes dan rekaman lengkap.
- Temporal date + langkah event; future evidence diblokir; stage/ACV/outlets snapshot hidden saat historis.
- Deal Universe: planet CSS, nebula/stars statis, klik zoom, fokus satu hop dan daftar relasi.
- Context Explorer: identitas/ID/tanggal, incoming/outgoing, penjelasan edge, navigasi neighbor dan tab bukti.
- Catalogue manual empat pendekatan historis dengan ActionOccurrence bersumber dari decision_log; semua requires_validation, outcome null.
- Pilih 2–4, server validation, Score/rank/Choice terpisah, stale guard, unavailable yang eksplisit.
- Copilot shared chat dengan scope tanggal/deal asli, pesan unavailable untuk provider yang belum ada.
- Desktop tiga kolom, evidence drawer untuk layar sedang, tabs mobile, dialog keyboard/focus return.

## Struktur

```text
src/app/             Next routes + same-origin API adapter/proxy
src/components/      Workspace, graph, evidence, comparison, Copilot
src/lib/contracts.ts DTO FE–BE
src/lib/domain.ts    Date/selection/score validation
src/lib/client.ts    Typed API client
src/lib/server/      Projection CSV/JSONL read-only
public/brand/        Logo/favicons GSM resmi
tests/               Domain regression + browser flows
```

## Hubungkan BE

Salin .env.example ke .env.local, isi RELIO_DATA_MODE=api dan RELIO_API_URL=http://127.0.0.1:8080.
Restart Next. Semua endpoint mengikuti /api dari PRD; lihat ../backend/README.md.
API keys dan implementasi JEV tetap di backend.

## Validasi

```powershell
npm run typecheck
npm test
npm run build
npm run test:e2e
```

Playwright menggunakan Chrome lokal. Atur RELIO_BROWSER_PATH ke executable Chromium yang tersedia.
Contoh PowerShell: $env:RELIO_BROWSER_PATH="C:\Program Files\Google\Chrome\Application\chrome.exe".
Production build terpisah pada .next-production supaya tidak bertabrakan dengan server dev; npm start menjalankan build tersebut.

## Batas implementasi saat ini

Backend Go/JEV, retrieval Copilot, Neo4j, ekstraksi catalogue otomatis, graph Contract/UsageAggregate/Decision lengkap, temporal employment resolution penuh, dan kit evaluasi A/B/C masih pekerjaan integrasi berikutnya. Coverage lokal merupakan proyeksi awal dan perlu review kualitas BE; tidak menjadi hasil final JEV. Outcome/consent tidak disimpulkan dari kemiripan.

Katalog manual tidak diklaim otomatis relevan dengan setiap deal. Mode api harus memakai DTO shared atau memperbarui kontrak bersama FE.

Hasil verifikasi implementasi: build production + TypeScript lulus, 5 domain tests dan 15 browser tests lulus. Lihat [QA aplikasi](../docs/frontend/qa/README.md) dan [status integrasi](../docs/frontend/Relio_FE_Implementation_Status_v1.0.md).

Panduan framework diperiksa dari docs yang disertakan Next.js terpasang. Referensi: [instalasi Next.js](https://nextjs.org/docs/app/getting-started/installation), [accessibility React Flow](https://reactflow.dev/learn/advanced-use/accessibility).
