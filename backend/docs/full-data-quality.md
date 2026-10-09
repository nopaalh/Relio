# Full preparation integrity

Source parser verifies all 15 hashes and row counts against source-audit-snapshot.json, native/composite duplicate keys, required audited headers, all nonblank native FKs, calendar/month values and nonnegative numeric source fields. Package limit native `tanpa batas` remains a string, never zero.

Cross-field checks include outlet/account alignment, decision native deal/account/evidence scope and chronology, reply scope/time, company-internal scope and prohibition of prospect daily usage. Native employee/contact meeting IDs must exist.

Known source limitations preserved: stale C01 champion versus K017 current P01 employment, overlapping employment intervals, unmatched historical Rina email, account-only interactions/decisions and retrospective decision fields. No source files modified.

First full preparation before action extraction: 229,627 source rows; 22 deals, 45 accounts, 4,320 domain nodes, 7,101 edges, 1,284 events, 19,811 evidence units. Action extraction and extra integrity fixes may change graph counts; final manifest/report is authoritative. Tests cover P04 three native interactions, P05 zero native interactions, native C23 decision links, deterministic artifacts and cutoff/missingness-aware usage.

Final preparation (before live acceptance): 33,225 typed records,2,914 digest partitions,480 source backing containers. Domain:4,537 nodes /7,535 edges /1,284 events /19,811 field evidence /22deals /24occurrences /12templates. All226,300daily usage rows retained. Serialized artifact file80,921,871bytes (compact JSON plus newline). Two full CLI preparations byte-identical; generated version full:d8e235c9a52b20fc6a16711f292744f3a06ff52096297ca3805e9f99dcb05a40. Raw profile emails no longer create inferred historical/foreign native node associations.

Generated artifact.json/manifest.json are ignored local preparation outputs; re-create via `go run ./tools/preparefull` from backend. Full Aura publication and idempotent rerun verified2026-10-10; exact live measurements/limits are in full-context-o2-handoff.md.
