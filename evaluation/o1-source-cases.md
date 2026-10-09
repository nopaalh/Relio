# O1 source-grounded evaluation cases

These are deterministic source/contract acceptance facts, not LLM/JEV ground truth or business-outcome predictions.

| Case | Native sources | Expected behavior |
|---|---|---|
| P04 native mapping | crm_deals.csv DL-004/account_id=P04 | Deal ID differs from account ID; historical stage unknown |
| P04 dated interactions | interactions.jsonl I0284/I0314/I0335 |2 interactions9/1;3 from9/22, each independent field proof; no future reply/thread leakage |
| P05 missing context | crm_deals.csv DL-005 created9/26, interactions roster |Before creation NotVisible; no invented interaction/action to reach comparison minimum |
| P03 created9/15 | crm_deals.csv DL-003 |9/1 NotVisible, not NotFound |
| C23 separate opportunities | crm_deals.csv DL-006/DL-007; decision_log.csv native deal_id |Do not infer deal links for account-only interaction; no future lost/won outcome from retrospective reason |
| Discount approval versus application | decision_log.csv, contracts_billing.csv explicit decision_id FK |Requester/approver distinct; approved occurrence not application/new permission;>10% policy evaluated by O2 with historical role uncertainty |
| Historical identity | contact_employment_history.csv K017; raw email participants |Employment intervals scoped, inclusive source end converted to exclusive bound; unmatched old email unresolved, not verified alias |
| Partial usage | product_usage_daily.csv date/outlet keys |Filter stored daily rows<=as_of; blank offline tracked missing, not zero; distinct outlets, no prospect-analog attribution |
| Roadmap/support snapshot | features.csv, bugs.csv, releases.csv, support_tickets.csv |Dated anchors/events visible, snapshot status/roadmap/current fields absent historically |
| Selected actions | native decision proof + primary interaction isi matching versioned rule |Future/foreign/invalid ID atomic failure; exact evidence bundle; historical keyword mention not unresolved gate or consent |

Acceptance implementations: repository/full_repository_test.go, full_actions_test.go, full_usage_test.go, full_integration_test.go and tools/preparefull tests. Live acceptance is read-only/explicit opt-in. No evaluation metric or commercial outcome is invented here.
