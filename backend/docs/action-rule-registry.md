# O1 action registry — relio-actions-v1 / relio-relevance-v1

Deterministic source-backed rules, not model findings or commercial authorization. Reviewed taxonomy is deliberately limited to decision_log.csv. Training/demo interactions remain graph events; they are not silently promoted to action candidates. No delivered-reference occurrence or reference consent is invented from a buyer request.

## Extraction

| Native type/value | Template | Occurrence/status |
|---|---|---|
| diskon, integer percentage 0–100 | action:discount:{percentage*100}bps | occ:decision:{decision_id} |
| pengecualian, value explicitly contains Starter and pilot | action:starter_pilot | Same native decision locator |
| janji_fitur | action:feature_promise | Same native decision locator |
| eskalasi | action:escalation | Same native decision locator |

Disetujui → approved; Ditolak → rejected; Menunggu → requested (pending decision, not granted approval). Unknown type/value/status is excluded, never guessed. Existing source requestor and approver are distinct participants/REQUESTED_BY and DECIDED_BY edges. Approval does not prove applied: contract application is a separate native FK and snapshot graph fact. Outcome remains unknown without dated support; retrospective alasan/status_janji do not become historical outcome.

Each occurrence links the existing decision event, actor/approver native IDs where supplied, date/status/type/value proofs and native deal ID only when supplied. Policy exposes integer BPS and the occurrence's decision status; historical approver role, new-request authorization, reference consent and outcome are not inferred. Discount >10% requires VP approval under application policy; a previous approved occurrence is not approval of a new request. Suitability and policy enforcement remain O2.

## Relevance

Visible primary-account interactions.jsonl field isi provides the contextual evidence. Literal case-insensitive keyword rules:

| Template type | Context keywords |
|---|---|
| discount | diskon, harga, budget, bujet, mahal |
| starter_pilot | pilot, uji coba |
| escalation | eskalasi |
| feature_promise | janji fitur |

These prove only a matching historical account-context mention, not that a gate remains unresolved, that an action is suitable, or that consent exists. No fallback menu. Each candidate must also have at least one visible occurrence with a dated event, status proof and source evidence. Primary native deal occurrence → current_deal; primary account-only occurrence → account_context; separately authorized foreign account → analog_precedent. Other native deals of the same account do not count as this deal.

Candidates sorted by template ID, actual total count, default/max50. Selected 1–4 unique valid IDs are resolved atomically with exact supporting field evidence; O2 comparison requires2–4. Rule version must be explicit. Zero/one candidate is legitimate. Optional MaxSelectedEvidenceBytes is not a frozen shared budget; zero imposes no artificial cap.

Prepared coverage before live publication: 24 occurrences /12 templates from30 native decisions. No claim that this is an exhaustive action ontology. Adding extraction/relevance rules requires a new mapping/dataset version before publication.
