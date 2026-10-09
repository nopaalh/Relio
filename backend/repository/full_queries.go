package repository

import (
	"github.com/nopaalh/Relio/backend/models"
	"sort"
)

type fullQuery struct {
	Version                         string
	Accounts, Analogs, Deals, Kinds []string
	Company                         bool
	AsOf                            models.Date
	Operation                       string
	LookupID                        string
	LookupKind                      string
}
type fullSelection struct {
	SourcePartitions []FullSourcePartition
	Manifest         FullManifest
	Partitions       []FullPartition
	Records          []FullRecord
}

func fullQueryFor(s models.SnapshotContext, operation string) fullQuery {
	accounts := append([]string{}, s.Access.AllowedAccountIDs...)
	analogs := []string{}
	if operation == "actions" {
		analogs = append(analogs, s.Access.AllowedAnalogAccountIDs...)
		accounts = append(accounts, analogs...)
	}
	accounts, _ = canonicalIDs(accounts)
	sort.Strings(accounts)
	kinds := []string{"deals", "nodes", "edges", "events", "evidence"}
	if operation == "actions" {
		kinds = append(kinds, "templates", "occurrences")
	}
	return fullQuery{Version: s.DatasetVersion, Accounts: accounts, Analogs: analogs, Deals: append([]string{}, s.Access.AllowedDealIDs...), Kinds: kinds, Company: s.Access.AllowCompanyEvidence, AsOf: s.AsOf, Operation: operation}
}

const fullManifestHeaderQuery = `MATCH (m:RelioFullManifest {version:$version}) RETURN m.ready AS ready,m.hash AS hash`
const fullSourceQuery = `MATCH (p:RelioFullSource {version:$version,id:$id}) WHERE p.account_id IN $accounts AND p.available_from <= $as_of RETURN p.payload AS payload`
const fullManifestPayloadQuery = `MATCH (m:RelioFullManifest {version:$version}) RETURN m.payload AS payload`

// Scope lookup is indexed; membership avoids scanning all records for an array predicate.
const fullRecordsQuery = `UNWIND $scopes AS scope MATCH (s:RelioFullScope {version:$version,scope:scope})-[:HAS_RECORD]->(r:RelioFullRecord)
WHERE r.kind IN $kinds AND r.available_from <= $as_of
AND (r.deal_id = '' OR r.deal_id IN $deals OR r.account_id IN $analogs)
RETURN DISTINCT r.envelope AS envelope
UNION
MATCH (r:RelioFullRecord {version:$version,kind:$lookup_kind,id:$lookup_id})
WHERE $lookup_id <> '' RETURN r.envelope AS envelope`
const fullPartitionsQuery = `UNWIND $ids AS id MATCH (p:RelioFullPartition {version:$version,id:id}) RETURN p.payload AS payload`
