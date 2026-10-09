# Full context mapping — O1

Mapping-v1 processes all 15 audited files once offline; no runtime CSV, ETL pipeline or refresh. All native source values remain in field evidence; snapshot-only values available at 2026-10-01.

- CRM accounts/deals/contacts/employees: source IDs and nullable commercial facts; no historical stage/owner inference.
- Employment: composite locator all six native fields; SourceRecordID empty; WORKED_AT effective [start,end+1), parallel roles preserved.
- Interactions: native account, meeting participants and reply edges; no automatic deal FK or verified historical email alias. Current email candidates unresolved.
- Decision: native date/type/value/status/actor/deal links; alasan/status_janji conservative snapshot-only to prevent retrospective outcome leakage.
- Contract: creation/native identity anchors at mulai; mutable terms/discount/decision link snapshot-only. No date/amount matching join.
- Ticket/bug/release/feature: native IDs, source references and creation/release anchors; current support/status/roadmap fields snapshot-only.
- Outlet: mapping snapshot-only. Daily usage rows stored in account/month containers with native (tanggal,outlet_id) keys and file checksum, not graph transaction nodes. Container grouping reduces DB nodes without losing daily precision.
- Usage aggregates: completed-month observed sums and unique outlets; missing offline count explicit. Partial-month raw contributions remain date-selectable backing evidence, never full-month totals attributed early.
- Feature usage: composite (bulan,account_id,feature_id); available month end, absent combinations not zero.

Graph JSON is not the source CSV. Aggregate evidence is labeled derived and records rule/basis, source checksum, account/month and observed coverage. It does not invent a native aggregate source_record_id. Template/candidate relevance is rule output, not source truth or consent.
