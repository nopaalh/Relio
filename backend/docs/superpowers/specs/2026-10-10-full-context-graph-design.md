# Relio O1 — Full Context Graph Design

Tanggal: 2026-10-10. Status: desain arah disetujui dalam chat; spec tertulis ini menunggu review user sebelum implementation plan. Bukan laporan implementasi selesai.

## Tujuan dan batas tanggung jawab

Selesaikan scope Orang 1: persiapan database statis dari seluruh 15 file kompetisi, graph temporal/provenance, query deal/graph/timeline/evidence, action candidates dan selected-action evidence, tests serta handoff ke Orang 2. Dataset tetap FIX. Arahan terbaru user menggantikan kewajiban pipeline ETL dalam PRD/development plan lama: persiapan offline sekali, tanpa connector, scheduler, refresh, CSV startup atau CSV request.

Go API dan Neo4j Aura dipertahankan. Package langsung di bawah backend; tidak membuat internal/ atau cmd/. Routes/controllers/services, DTO HTTP, scoring, JEV, cache, Copilot, auth server dan frontend tetap O2. Handoff bukan klaim JEV/chatbot/UI sudah terintegrasi.

Acuan: PRD Final MVP v1.0 di root HeketonPens, READMEHack.md, source-mapping.md, dataset-audit.md, data-repository-contract-response-o2.md dan interface Go aktual. Shared contract tetap candidate sampai dibekukan bersama O2.

## Kondisi awal dan kompatibilitas

P04/DL-004 sudah dimuat ke Aura, 46 typed records dan 8 adjacency links; acceptance read-only tiga snapshot PASS. Namespace tersebut dipertahankan. Loader P04 sudah dipush pada 3f4ee19.

Adapter P04 membaca seluruh namespace kecil dan memiliki asumsi P04/DL-004. Implementasi full harus menghilangkan asumsi itu tanpa mengubah signature DealFactsRepository, GraphRepository, TimelineRepository, EvidenceRepository atau ActionRepository. Legacy DealRepository tidak diam-diam diganti atau diberi lossy bridge. O2 memilih adapter/dataset version pada wiring-nya.

Gunakan namespace dan schema version full yang berbeda. IDs sumber tetap canonical; ID turunan menggunakan tuple deterministik yang mencakup jenis, relasi, locator dan rule version. Dataset version diturunkan dari hashes semua sumber dan versi mapping/taxonomy/temporal. P04 tests tetap menjadi regression suite.

## Coverage sumber

- CRM: seluruh account, deal, contact dan employee; snapshot facts per field, owner dan champion sebagai klaim snapshot dengan bukti, bukan historical authorization.
- Employment: seluruh interval, organisasi eksternal tanpa invented account ID, job-role evidence; overlap dan stale champion dipertahankan sebagai limitations. Interval efektif [from,to), source end inclusive dikonversi +1 hari dengan basis eksplisit.
- Interactions: setiap record, native participants, sender/targets raw, reply native, subjek/isi evidence. Account-only tidak diberi deal ID secara otomatis. General internal record memerlukan AllowCompanyEvidence.
- Decisions dan contracts: event/relasi native, requester/approver, request/decision/application facts terpisah. Native FK hanya dipakai bila tersedia. Persetujuan analog tidak berlaku pada request aktif.
- Outlets, daily usage dan monthly feature usage: seluruh row diproses dan dipertanggungjawabkan; output graph berupa UsageAggregate, bukan 226.300 node transaksi individual, sesuai PRD.
- Support tickets, bugs, releases dan features: enrichment bertipe dengan native links dan temporal limitations; tidak infer retrospective status/roadmap sebagai fakta lampau.

Counts 229.627 records dari audit sebelumnya adalah baseline yang harus diverifikasi ulang dari sumber/hashes, bukan angka output graph wajib. Setiap source row harus tercatat sebagai mapped, aggregate contribution atau explicit diagnostic; tidak silently skipped.

## Representasi usage dan provenance

Simpan aggregate contribution per account/hari dengan transaksi, jumlah outlet observed, versi aplikasi dan offline known/missing counts. Monthly view hanya memakai tanggal <= as_of; month belum lengkap diberi observed-through-cutoff, bukan total full month. Monthly feature record available paling awal akhir bulan sumber. Missing combination/offline blank bukan zero.

Simpan backing source rows dalam partition account/hari dengan locator composite asli (tanggal,outlet_id), file checksum, row/field checksum dan aggregation rule. Ini evidence storage, bukan node transaksi individual atau fallback membaca CSV saat runtime. Bukti summary menyebut cakupan/rule, tidak mengklaim aggregate sebagai kutipan source row. Detail evidence harus tetap bisa di-resolve dari data tersimpan.

Person label/value dari snapshot sebelum 2026-10-01 tetap unknown/snapshot_only kecuali ada bukti bertanggal. Meeting native participant ID boleh verified; historical current-email match hanya candidate, bukan verified alias. Tidak mengarang recorded_at atau source_record_id: sumber tanpa native ID memiliki SourceRecordID kosong (tipe existing string), RecordIDKind composite, dan SourceKey tuple asli. EvidenceID/entity ID turunan berlabel derived; tidak ditampilkan sebagai ID sumber native.

## Query adapter full

Index namespace, record kind/ID, scope account/deal, availability date dan event date. Runtime query parameterized, AccessModeRead, driver/Cypher tetap privat. Tidak membaca semua akun, semua evidence, semua usage atau seluruh digest inventory pada tiap request.

Manifest readiness didapat dari validasi preparation/load seluruh namespace. Payload hash dibandingkan registry terikat manifest pada partition yang dipilih; aggregate partition root digests mengikat namespace tanpa memuat seluruh registry per request. Hash bukan signature atau credential.

List/detail hanya account/deal primary authorized. Graph/timeline mengambil konteks primary serta traversal depth 1 default, maksimum 2; graph maksimum 150 nodes/300 edges. Paging maksimum/default 50, urutan stabil, cursor terikat context/filter/scope. Focus selalu mendapat slot bila query sukses, tidak membuka event future/foreign. Semua endpoints/ref returned harus authorized dan dapat di-resolve pada snapshot sama; truncation eksplisit.

Analog accounts hanya dibaca bila izin eksplisit dan operation relevan; tidak memberikan akses primary secara otomatis. Evidence company-wide perlu flag eksplisit. Scope kosong ditolak. Cross-account employment/people tidak boleh membocorkan label/refs/source dari akun yang tidak diizinkan.

Temporal semantics mengikuti as_of hari kalender Asia/Jakarta, maksimum 2026-10-01. Snapshot-only commercial facts historical null. Retrospective fields keputusan/support/roadmap hanya available saat snapshot kecuali span/field punya bukti tanggal lebih awal; memfilter tanggal record saja tidak cukup. Seluruh thread tidak pernah dikembalikan dengan future replies.

## Action catalog dan retrieval

Audit seluruh interactions/decisions/contract source sebelum menetapkan registry. Taxonomy/relevance registry berversi mencatat rule, source locator/quote dan limitations. Hanya tindakan Sales/company yang benar-benar terdokumentasi menjadi occurrence; buyer request dapat menjadi relevance evidence, bukan otomatis completed company action.

Template minimal satu occurrence berbukti. Status requested/proposed/offered/approved/rejected/applied/completed/pending tidak disatukan. Outcome null tanpa bukti observed outcome bertanggal; Won sesudah action bukan causal success. Actor/targets ambiguous tetap unresolved.

ReadActionCandidates mengembalikan template dengan precedent visible/authorized dan relevance reason yang memiliki bukti target deal/account-context dan precedent. Current-deal occurrence dibedakan dari account-only context dan analog_precedent. Relevance bukan score/suitability. Jika tidak ada rule berbukti, candidate 0; tidak memakai fallback semua templates atau padding untuk comparison.

ReadSelectedActions memvalidasi unique IDs terhadap candidate set dan rule version yang sama, lalu mengembalikan evidence bundle lengkap secara atomik. Satu ID boleh untuk inspeksi; O2 mengatur 2–4 untuk compare. Unknown/invalid selected ID, scope/time mismatch, atau cap terlampaui menghasilkan typed error, bukan partial success.

PolicyFacts melekat pada occurrence/request: discount BPS, native decision/application links, approval, approver dan consent target/scope. Blank bukan false/rejected/approved. >1000 BPS memerlukan VP policy di O2; approver role snapshot bukan historical verified VP. Approval lama/analog dan reference suitability tidak menjadi izin baru. Bundle bytes/counts diukur; cap final integrasi model tetap perlu kesepakatan O2, bukan angka rekaan yang dibekukan O1.

## Preparation/load dan keamanan

Persiapan offline memverifikasi 15 hashes, FK, dates, duplicate keys, source counts, stable IDs, evidence closure, nulls dan aggregates. Artefak deterministic, tidak mengubah dataset. Satu loader eksplisit melakukan preflight target/version/hash, schema IF NOT EXISTS, writes terbatas namespace baru dan readiness terakhir setelah validasi.

Tidak DELETE/overwrite namespace P04, tidak auto-switch .env/server ke versi baru. Batch load full tidak harus satu transaksi besar; readiness tetap false saat incomplete. Rerun same namespace idempotent; mismatch berhenti. Before live load, laporkan projected nodes/relationships/storage dan cek kapasitas instance; jangan upgrade berbayar atau menghapus data tanpa izin.

Credential hanya file lokal ignored/process environment. Tidak mencetak password, raw driver errors atau source payload ke log. Runtime adapter read-only; preparation tool terpisah dari server lifecycle. O2 menentukan HTTP error mapping, auth, logging dan model metadata/cache.

## Acceptance dan handoff

1. Seluruh 15 source files/hashes/counts terverifikasi; every row accounted; 22 deals/45 accounts dari audit dikonfirmasi, P01–P05 native mapping diuji.
2. Native IDs/ref closure, employment intervals, external organization dan ambiguous identities benar; source evidence bisa dibuka dan diverifikasi ke field/span.
3. Snapshot switch konsisten; future refs/end dates/replies/retrospective outcome tidak bocor; current commercial values historical null.
4. Scope allow/deny primary/analog/company, graph caps/focus/paging/cursor, selected batch atomic dan typed errors diuji.
5. Usage aggregate sums sesuai source, unique outlets benar, partial month jelas, blank offline tidak jadi zero, prospek tidak diberi usage pelanggan.
6. Kandidat hanya berbukti/relevan, jumlah aktual 0/1 diterima, request/approval/application/consent terpisah; preseden future tidak masuk.
7. Full suite, vet/race, deterministic preparation, duplicate-free rerun dan Aura live PASS sebelum klaim selesai. Query latency/bundle measurements dilaporkan actual; target bukan hasil pengukuran.
8. Handoff O2: constructor/config/dataset version, registry versions, tested examples dan acceptance report, error contract serta batas coverage/unknown. Evaluation factual cases berbasis source ditambahkan untuk graph/action/policy; tidak mengarang skor/hasil peserta/JEV.

Tidak mengklaim seluruh MVP selesai dari keberhasilan O1. Gerbang selanjutnya setelah review spec: tulis implementation plan, user review plan, eksekusi native yang sudah dipilih user sebelumnya.
