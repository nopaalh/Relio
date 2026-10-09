# Relio Backend

Go backend dengan struktur package langsung di bawah `backend/`.

## Dokumentasi pengembangan

- [Arsitektur backend berdasarkan PRD Final MVP v1.0](docs/architecture.md): keputusan produk, kondisi kode, context graph, assessment, action comparison, API, keamanan, dan validasi.
- [Pertanyaan terbuka dan rekomendasi v1](docs/open-questions.md): bagian yang telah dipertegas PRD versus keputusan teknis yang masih terbuka, beserta alasan, risiko, dan owner usulan.
- [Rencana pengembangan dua orang](docs/development-plan.md): O1 fokus context graph/data, O2 fokus aplikasi backend; backlog, batas file, kontrak handoff, milestone, dan kriteria selesai.

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

Endpoint yang sudah tersedia: `GET /healthz`.

`DealRepository` dan `DealService` masih berupa fondasi kontrak untuk fitur deal berikutnya; belum ada adapter dataset/database atau endpoint deal. Tidak ada data contoh yang disajikan seolah berasal dari dataset.
