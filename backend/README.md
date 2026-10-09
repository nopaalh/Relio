# Relio Backend

Go backend dengan struktur package langsung di bawah `backend/`.

## Dokumentasi pengembangan

- [Arsitektur backend berdasarkan PRD Final MVP v1.0](docs/architecture.md): keputusan produk, kondisi kode, context graph, assessment, action comparison, API, keamanan, dan validasi.
- [Pertanyaan terbuka dan rekomendasi v1](docs/open-questions.md): bagian yang telah dipertegas PRD versus keputusan teknis yang masih terbuka, beserta alasan, risiko, dan owner usulan.
- [Rencana pengembangan dua orang](docs/development-plan.md): O1 fokus context graph/data, O2 fokus aplikasi backend; backlog, batas file, kontrak handoff, milestone, dan kriteria selesai.
- [Jawaban review kontrak dari O2](docs/data-repository-contract-response-o2.md): jawaban F01–F08 dan urutan integrasi; masih memerlukan konfirmasi O1.
- [Target kontrak context API — OpenAPI 3.1](docs/context-api.openapi.json): draft DTO dan query untuk deal facts, graph, timeline, dan evidence; bukan deskripsi endpoint yang semuanya sudah berjalan.

Dokumen membedakan kebutuhan PRD, kode yang sudah ada, default teknologi bersyarat, dan usulan teknis. Database/framework aktual, provider login, serta verifikasi live API model belum dianggap selesai. Ownership frontend/UI E2E tetap perlu diputuskan; API backend saja bukan MVP lengkap.

## Struktur package

```text
backend/
├── main.go
├── go.mod
├── routes/
├── controllers/
├── services/
├── repository/
├── models/
└── docs/
```

## Tanggung jawab package

- `routes`: mendaftarkan endpoint dan menghubungkannya dengan controller.
- `controllers`: menangani request/response HTTP; tidak berisi business logic atau query data.
- `services`: aturan bisnis dan orkestrasi repository.
- `repository`: kontrak dan adapter akses data; tidak bergantung pada HTTP.
- `models`: tipe data domain dan response.
- `main.go`: entry point dan dependency wiring.

Alur request: `routes → controllers → services → repository`. `models` dipakai oleh layer yang memerlukan tipe data, bukan merupakan langkah eksekusi terakhir.

## Menjalankan

Dari folder `backend/`, gunakan Go 1.22 atau lebih baru:

```sh
go run .
go test ./...
```

Server memakai `ADDR` jika tersedia, dengan default `:8080`.

## Target kontrak context API

`docs/context-api.openapi.json` adalah **target draft**, belum kontrak bersama yang final. File JSON bisa dibuka di tooling yang mendukung OpenAPI 3.1; tidak perlu server Swagger atau dependency baru di aplikasi.

Kontrak fase context-read mencakup `/healthz`, list/detail deal, graph, timeline, dan evidence. Ranking overview berbasis assessment, actions, dan endpoint model belum masuk fase ini. Setiap operasi memakai penanda `x-relio-implementation`:

- `implemented`: health scaffold yang sudah berjalan.
- `legacy-scaffold`: dua endpoint deal masih memakai model/response legacy dan snapshot tetap; envelope facts pada OpenAPI adalah target migrasi, bukan respons saat ini.
- `planned`: graph, timeline, dan evidence belum didaftarkan sebagai route runtime.

Target mengatur nullable facts dengan evidence, metadata/context bersama, date-only seluruh hari Asia/Jakarta, cutoff 2026-10-01, bounds/pagination, serta safe error response. Default graph depth 1 (maksimum 2), maksimal 150 nodes/300 edges, dan page size maksimal 50. Aturan tersebut belum dianggap diterapkan pada endpoint planned.

O1 tetap memiliki domain/interface Go dan adapter. O2 menyiapkan DTO/HTTP boundary terhadap kontrak ini setelah shared types dan keputusan inti dikonfirmasi. Tidak membangun model graph atau query sumber kedua, dan tidak melayani JSON audit/fixture sebagai data bisnis.

`routes/openapi_test.go` memeriksa konsistensi draft, required fields, bounds, dan local references. Tes ini **bukan validator OpenAPI lengkap**, pengujian handler planned, atau bukti database siap. Jalankan pemeriksaan spesifik dari `backend/`:

```sh
go test ./routes -run TestContextOpenAPI -v
```

## Endpoint yang tersedia

| Method | Route | Status saat ini |
|---|---|---|
| GET | `/healthz` | Health scaffold; bukan pemeriksaan kesiapan database/model. |
| GET | `/api/deals` | Controller/service tersedia; mengembalikan `503` sampai adapter database dipasang. |
| GET | `/api/deals/{deal_id}` | Controller/service tersedia; mengembalikan `503` sampai adapter database dipasang. |

### Snapshot dan kontrak sementara

Endpoint deal memakai snapshot tetap `2026-10-01`. Parameter `as_of` boleh tidak diberikan atau bernilai tepat `2026-10-01`; nilai kosong, berulang, invalid, atau tanggal lain mengembalikan `400`. Tanggal diteruskan sebagai token `time.Time` pada `2026-10-01T00:00:00Z`, **bukan keputusan final tentang timezone/batas hari query historis**. Historical queries menunggu kontrak context graph dari orang pertama.

Ketika adapter tersedia, list mengembalikan JSON array `models.Deal` dan detail mengembalikan satu objek. Bentuk model/DTO ini masih scaffold, bukan kontrak final untuk nullable facts, provenance, atau historical context. Array kosong `[]` hanya untuk query sukses tanpa hasil; adapter belum terpasang tidak dianggap hasil kosong.

Error deal memakai JSON `{ "code": "...", "message": "..." }`:

- `400`: ID kosong atau snapshot tidak didukung.
- `404`: repository mengembalikan `repository.ErrDealNotFound`.
- `405`: method bukan GET, dengan header `Allow: GET`.
- `503`: adapter belum terpasang atau mengembalikan `repository.ErrDealDataUnavailable`.
- `500`: kegagalan lain, tanpa membocorkan detail internal repository.

### Handoff adapter database statis

Tidak ada ETL, penarikan dataset, database driver baru, atau data demo yang disajikan sebagai data asli. Orang pertama menyediakan implementasi `repository.DealRepository` dengan `List`/`FindByID`; pasang instance tersebut menggantikan `services.NewDealService(nil)` di `main.go`. Gunakan atau wrap sentinel error repository agar HTTP mapping tetap bekerja.

Mock/stub hanya berada pada tests. Startup saat ini sengaja menghasilkan `503` pada endpoint deal, bukan klaim koneksi database berhasil.

Contoh request dari terminal Windows setelah server dijalankan:

```sh
curl.exe -i "http://localhost:8080/api/deals"
curl.exe -i "http://localhost:8080/api/deals/DL-001?as_of=2026-10-01"
```

Pengujian:

```sh
go test ./...
go vet ./...
```

Graph, timeline, evidence, scoring, JEV, action comparison, dan Copilot belum diimplementasikan pada langkah fondasi endpoint ini.
