# Relio Backend

Go backend dengan struktur package langsung di bawah `backend/`.

## Dokumentasi pengembangan

- [Arsitektur backend berdasarkan PRD v0.4](docs/architecture.md): keputusan yang sudah ditetapkan, kondisi implementasi saat ini, komponen target, alur data/graph, API, dan rencana validasi.
- [Pertanyaan terbuka dan rekomendasi](docs/open-questions.md): keputusan yang belum final beserta saran, alasan, risiko, dan tahap ketika keputusan dibutuhkan.

Dokumen membedakan kebutuhan PRD, kode yang sudah ada, dan usulan teknis. Database, provider login, serta detail integrasi model belum dianggap sebagai keputusan final.

## Struktur package

```text
backend/
├── main.go
├── go.mod
├── routes/
├── controllers/
├── services/
├── repository/
└── models/
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
