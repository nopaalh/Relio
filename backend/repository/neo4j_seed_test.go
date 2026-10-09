package repository

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"os"
	"strings"
	"testing"
)

func preparedP04(t *testing.T) storedContext {
	t.Helper()
	b, err := os.ReadFile("../database/neo4j/context-p04.json")
	if err != nil {
		t.Fatal(err)
	}
	var raw storedContext
	if err := json.Unmarshal(b, &raw); err != nil {
		t.Fatal(err)
	}
	return raw
}
func TestNeo4jSeedP04SourceGrounding(t *testing.T) {
	raw := preparedP04(t)
	if raw.Manifest.Ready || raw.Manifest.SchemaVersion != "relio-neo4j-p04-v1" || len(raw.Deals) != 1 || raw.Deals[0].Value.DealID != "DL-004" || raw.Deals[0].Value.AccountID != "P04" {
		t.Fatal("wrong slice/false readiness")
	}
	ids := []string{"event:interaction:I0284", "event:interaction:I0314", "event:interaction:I0335"}
	for i, e := range raw.Events {
		if e.EventID != ids[i] || len(e.DealIDs) != 0 || e.ScopeKind != "account" || e.SourceTimestamp != nil {
			t.Fatal("invented linkage/time")
		}
	}
	if len(raw.Events) != 3 || raw.Events[2].Summary.Value == nil || *raw.Events[2].Summary.Value != "Pak Bagus, Direktur Utama kami minta rekomendasi dari pengguna yang mirip dengan kami sebelum tanda tangan. Kami tunda dulu sampai ada referensi." {
		t.Fatal("source quote changed")
	}
	if raw.Events[2].Actors[0].NodeID != nil || len(raw.Events[2].Actors[0].CandidateNodeIDs) != 0 {
		t.Fatal("historical email promoted")
	}
	if raw.Events[0].Participants[0].NodeID == nil || *raw.Events[0].Participants[0].NodeID != "contact:K065" {
		t.Fatal("native participant lost")
	}
}
func TestNeo4jSeedReferences(t *testing.T) {
	raw := preparedP04(t)
	if err := validateStoredContext(raw); err != nil {
		t.Fatal(err)
	}
	for _, e := range raw.Evidence {
		if len(e.Value.SourceChecksum) != 64 || e.Value.SourceRecordID == "" || e.Value.RecordIDKind == "" {
			t.Fatal("missing provenance")
		}
	}
	broken := preparedP04(t)
	broken.Edges[0].Value.Target = "missing"
	if validateStoredContext(broken) == nil {
		t.Fatal("dangling endpoint accepted")
	}
	broken = preparedP04(t)
	broken.Events[0].Summary.EvidenceIDs = []string{"missing"}
	if validateStoredContext(broken) == nil {
		t.Fatal("dangling fact accepted")
	}
}
func TestNeo4jSeedSafeAndDeterministic(t *testing.T) {
	for _, file := range []string{"schema.cypher", "seed-p04.cypher"} {
		b, err := os.ReadFile("../database/neo4j/" + file)
		if err != nil {
			t.Fatal(err)
		}
		u := strings.ToUpper(string(b))
		for _, bad := range []string{"DETACH DELETE", "DROP CONSTRAINT", "LOAD CSV"} {
			if strings.Contains(u, bad) {
				t.Fatal("unsafe seed")
			}
		}
	}
	b, _ := os.ReadFile("../database/neo4j/seed-p04.cypher")
	if !strings.Contains(string(b), "ON CREATE SET") || !strings.Contains(string(b), "manifest_hash") {
		t.Fatal("no immutable guard")
	}
}

func TestNeo4jSeedChecksActualPayloadAndAdjacency(t *testing.T) {
	b, err := os.ReadFile("../database/neo4j/seed-p04.cypher")
	if err != nil {
		t.Fatal(err)
	}
	q := string(b)
	// Inspect the published gate, not merely stored hash labels/counts.
	if !strings.Contains(q, "n.payload=x.payload") || !strings.Contains(q, "single(r IN links WHERE") || !strings.Contains(q, "r.source=x.source") || !strings.Contains(q, "r.type=x.type") {
		t.Fatal("readiness gate does not validate actual bytes/adjacency")
	}
}

func TestNeo4jManifestBindingRejectsPayloadHashReplacement(t *testing.T) {
	raw := preparedP04(t)
	rows := preparedNeo4jRows(t)
	if _, err := decodeNeo4jRecords(raw.Manifest, rows); err != nil {
		t.Fatal("valid sealed records rejected", err)
	}
	// Alter actual business payload AND its adjacent digest; namespace seal stays.
	for i := range rows {
		if rows[i].Kind == "events" {
			rows[i].Payload = strings.Replace(rows[i].Payload, "35 outlet", "36 outlet", 1)
			sum := sha256.Sum256([]byte(rows[i].Payload))
			rows[i].Hash = hex.EncodeToString(sum[:])
			break
		}
	}
	if _, err := decodeNeo4jRecords(raw.Manifest, rows); err == nil {
		t.Fatal("payload+hash change escaped manifest binding")
	}
	raw.Deals[0].Value.DealID = "DL-999"
	if validateStoredContext(raw) == nil {
		t.Fatal("renamed native anchor accepted")
	}
	raw = preparedP04(t)
	raw.Manifest.Records[0].Hash = strings.Repeat("0", 64)
	if validateStoredContext(raw) == nil {
		t.Fatal("changed registry under old manifest seal accepted")
	}
}
func preparedNeo4jRows(t *testing.T) []neo4jRecordRow {
	t.Helper()
	b, err := os.ReadFile("../database/neo4j/context-p04.json")
	if err != nil {
		t.Fatal(err)
	}
	var obj map[string]json.RawMessage
	if err = json.Unmarshal(b, &obj); err != nil {
		t.Fatal(err)
	}
	raw := preparedP04(t)
	rows := []neo4jRecordRow{}
	for _, kind := range []string{"deals", "nodes", "edges", "events", "evidence"} {
		var values []json.RawMessage
		if err = json.Unmarshal(obj[kind], &values); err != nil {
			t.Fatal(err)
		}
		for _, value := range values {
			sum := sha256.Sum256(value)
			hash := hex.EncodeToString(sum[:])
			id := ""
			for _, record := range raw.Manifest.Records {
				if record.Kind == kind && record.Hash == hash {
					id = record.ID
					break
				}
			}
			if id == "" {
				t.Fatal("prepared record lacks manifest binding")
			}
			rows = append(rows, neo4jRecordRow{Kind: kind, ID: id, Payload: string(value), Hash: hash})
		}
	}
	return rows
}
