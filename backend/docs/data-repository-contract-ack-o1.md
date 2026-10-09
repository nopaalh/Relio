# Relio — ACK O1 atas review kontrak O2

Tanggal: 10 Oktober 2026. Penulis: O1, context graph/data.

Status: **ACK prinsip dan batas tanggung jawab; deklarasi Go final masih menunggu review O2.** Ini bukan laporan adapter/database siap, hasil acceptance, atau persetujuan deployment.

Acuan: [draft O1](data-repository-contract-draft.md), [review O2](data-repository-contract-response-o2.md), dan [rencana implementasi tahap pertama](superpowers/plans/2026-10-10-context-contract-foundation.md).

## Keputusan yang diterima O1

| ID | ACK O1 | Gerbang yang tetap terbuka |
|---|---|---|
| F01 | Satu graph database bisnis statis, disiapkan sekali; runtime read-only. Neo4j/Aura tetap arah bersyarat, bukan engine/instance yang sudah dipilih. | Pilihan engine/instance, akses aman, schema/data siap, integration test nyata. |
| F02 | Tambahkan `DealFactsRepository` dan facts nullable. Pertahankan `DealRepository`, `models.Deal`, serta consumer existing sampai migrasi eksplisit O2. Tidak membuat bridge yang menghilangkan unknown/provenance. | Review O2 atas deklarasi Go dan contract fixtures; migrasi DTO/service milik O2. |
| F03 | Date-only, seluruh hari Asia/Jakarta, maksimum 2026-10-01; event pada cutoff termasuk. Employment `selesai` terisi menjadi hari terakhir inklusif lalu +1 hari untuk interval internal half-open. | Ini konvensi tim, bukan verifikasi makna end-day sumber. Raw date disimpan; overlap tidak diperbaiki otomatis. Tests boundary dan proyeksi historical diperlukan. |
| F04 | Account-only tetap account-scoped; **tidak ada single-deal fallback pada MVP awal**. Link deal hanya native atau reviewed rule berbukti. Ambiguity tidak dipilih menjadi identitas verified. | Registry identity/linkage bersumber; historical alias/role tidak ditebak dari CRM saat ini. |
| F05 | Scope dibentuk O2 di server; empty scope tidak memberi izin. Izin analog dan company evidence eksplisit, terpisah dari current-deal permission. | Gate/login dan allowlist demo aktual milik O2. Tidak membuka semua akun demi denominator atau preseden. |
| F06 | Candidate memerlukan occurrence dan relevance reason berbukti, visible dan authorized. Policy facts melekat pada request/occurrence; preseden bukan approval/consent. Unknown izin tidak sama dengan rejection atau executable. | Taxonomy/relevance registry perlu review sumber. Warning, suitability, scoring dan compare validation milik O2. |
| F07 | Depth default 1/max 2; maksimum 150 nodes, 300 edges; page default/max 50. Selected bundle lengkap atau error; satu selected ID boleh diinspeksi, 2–4 unik untuk compare milik O2. | Cap evidence menunggu ukuran bundle yang benar-benar diukur. Tidak membuat opsi tambahan atau memotong bukti diam-diam. |
| F08 | Terima Fact/meta/context/bounds, typed errors dengan Unwrap, dan selected batch atomik. Cause/internal refs tidak menjadi respons publik. | Nama/type Go konkret dan versi kontrak masih perlu review O2; DTO/tag/error HTTP final tetap milik O2. |

## Klarifikasi deklarasi yang perlu terlihat saat review Go

1. Usulan `AccessScope` menambahkan `AllowedAnalogAccountIDs` karena draft awal hanya memiliki allowlist account/deal dan company flag. Field baru ini menyatakan izin analog secara eksplisit; bukan alasan otomatis memilih suatu account sebagai analog, dan tidak memberi akses current-deal.
2. Versi untuk deklarasi awal diusulkan `context-contract-v0.1-candidate`. Bukan versi final bersama. Dataset version wajib berasal dari persiapan data; tidak diisi dengan string rekaan agar query tampak siap.
3. Domain types dan interface berada langsung di `models/` dan `repository/`. Tipe domain bukan otomatis DTO HTTP yang final. Arrays pada output valid harus non-nil; O2 memiliki serialization/public DTO checks.
4. Helper snapshot hanya menyatukan semantik tanggal, versi, dan scope. Hash context bukan token akses. Repository nyata tetap memeriksa snapshot/context, scope hasil dan setiap reference.
5. Tahap pertama tidak mengimplementasikan DB, query graph, projection historical seluruh domain, atau action relevance dengan fixture seolah data kompetisi. Acceptance adapter T01–T15 tetap gerbang berikutnya.

## Batas perubahan dan kondisi yang diperiksa

Working tree diperiksa pada main `57420ac` (merge foundation O2), sebelum perubahan dokumen ini. File existing `models/deal.go`, `repository/deal_repository.go`, `main.go`, routes/controllers/services tidak diubah oleh ACK ini. Main memiliki endpoint deal dan repository nil; belum ada adapter bisnis.

Skill planning mewajibkan review rencana implementasi sebelum product code. Karena itu ACK ini diserahkan bersama rencana yang spesifik; source baru belum ditulis. Review O2 atas nama/shape deklarasi tetap diperlukan sebelum consumer final dibekukan.

Percobaan baseline `go test ./...` dari backend pada sesi ini tidak berjalan: `go` tidak ditemukan di PATH lingkungan agent. Ini bukan test failure kode dan bukan hasil tests lulus. Tidak ada instalasi toolchain atau perubahan permission/config yang dilakukan.

## Handoff minimum O2

- ACK F02/F03/F04/F05/F08 di atas tersedia untuk melanjutkan review kontrak, tanpa memilih vendor DB.
- Review rencana tahap pertama dan tambahan scope analog. O2 dapat menyiapkan test-only stubs dan DTO/service design; jangan membuat source mapping kedua.
- Sesudah shared declarations tersedia dan direview: O1 mengerjakan adapter/data/temporal/evidence; O2 mengerjakan consumer, temporal/access HTTP validation dan wiring. Slice pertama tetap DL-004/P04 dengan interactions account-scoped.
- Dokumen draft lama dan arsitektur bersama tidak ditimpa atau ditandai final oleh ACK ini.

## Pembaruan pelaksanaan — 10 Oktober 2026

Sesudah user mereview rencana dan memberi "gas", implementasi tahap pertama dilakukan pada branch feat/context-contract-o1. User juga meminta instalasi Go; Go 1.27.0 resmi dipasang dan version command diverifikasi. Kendala PATH/cache awal di atas adalah catatan sesi sebelumnya, bukan status toolchain akhir.

Tipe domain, interface, typed errors, helper date/snapshot/scope/bounds dan fixtures DATA UJI tersedia untuk review O2. Tests lengkap, vet dan race tests lulus; rincian aktual ada pada [handoff Go](context-contract-go-handoff.md). Keberhasilan ini bukan acceptance database, approval consumer O2, atau penyelesaian MVP. Source legacy/API/draft/arsitektur bersama tetap tidak diubah; belum commit/push/merge.
