// Explicit one-off full namespace loader; never invoked by the API server.
package main

import (
	"context"
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"github.com/neo4j/neo4j-go-driver/v5/neo4j"
	"github.com/nopaalh/Relio/backend/models"
	"github.com/nopaalh/Relio/backend/repository"
	"net/url"
	"os"
	"sort"
	"strings"
	"time"
)

func parseConfig(text string) (repository.Neo4jConfig, error) {
	values := map[string]string{}
	allowed := map[string]bool{"NEO4J_URI": true, "NEO4J_USERNAME": true, "NEO4J_PASSWORD": true, "NEO4J_DATABASE": true}
	for _, line := range strings.Split(text, "\n") {
		line = strings.TrimSpace(strings.TrimPrefix(line, "\ufeff"))
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		k, v, ok := strings.Cut(line, "=")
		k = strings.TrimSpace(k)
		if !ok {
			return repository.Neo4jConfig{}, errors.New("invalid .env line")
		}
		if !allowed[k] {
			continue
		}
		if _, exists := values[k]; exists {
			return repository.Neo4jConfig{}, fmt.Errorf("duplicate config key: %s", k)
		}
		v = strings.TrimSpace(v)
		if strings.HasPrefix(v, "'") || strings.HasPrefix(v, "\"") {
			if len(v) < 2 || v[len(v)-1] != v[0] {
				return repository.Neo4jConfig{}, fmt.Errorf("unclosed quote: %s", k)
			}
			v = v[1 : len(v)-1]
		}
		values[k] = v
	}
	for k := range allowed {
		if values[k] == "" || values[k] == "PASSWORD_AURA_KAMU" || strings.Contains(values[k], "xxxxxxxx") {
			return repository.Neo4jConfig{}, fmt.Errorf("missing/placeholder config: %s", k)
		}
	}
	u, err := url.Parse(values["NEO4J_URI"])
	if err != nil || u.Scheme != "neo4j+s" || u.Host == "" || u.User != nil || u.RawQuery != "" || u.Fragment != "" {
		return repository.Neo4jConfig{}, errors.New("invalid NEO4J_URI")
	}
	return repository.Neo4jConfig{URI: values["NEO4J_URI"], Username: values["NEO4J_USERNAME"], Password: values["NEO4J_PASSWORD"], Database: values["NEO4J_DATABASE"]}, nil
}
func checkBudget(existingNodes, existingLinks, newNodes, newLinks, nodeBudget, linkBudget int64) error {
	if existingNodes+newNodes > nodeBudget || existingLinks+newLinks > linkBudget {
		return fmt.Errorf("projected DB size exceeds conservative configured budget; no writes performed")
	}
	return nil
}
func safeError(stage string, err error) error {
	var e *neo4j.Neo4jError
	if errors.As(err, &e) {
		return fmt.Errorf("%s failed (%s); raw error omitted", stage, e.Code)
	}
	return fmt.Errorf("%s failed; raw error omitted", stage)
}
func scopeIDs(r repository.FullRecord) []string {
	if len(r.AccountIDs) == 0 {
		return []string{"company"}
	}
	return r.AccountIDs
}
func run() error {
	artifact := flag.String("artifact", "database/neo4j/full/artifact.json", "reviewed generated artifact")
	expect := flag.String("expect-version", "", "explicit namespace version required for apply")
	apply := flag.Bool("apply", false, "write prepared full namespace")
	inspect := flag.Bool("inspect", false, "read-only preflight")
	nodeBudget := flag.Int64("node-budget", 50000, "conservative node ceiling, not a claim about subscription tier")
	linkBudget := flag.Int64("link-budget", 175000, "conservative relationship ceiling")
	flag.Parse()
	data, err := os.ReadFile(*artifact)
	if err != nil {
		return errors.New("cannot read generated full artifact")
	}
	var a repository.FullArtifact
	if json.Unmarshal(data, &a) != nil {
		return errors.New("malformed full artifact")
	}
	if err := repository.ValidateFullArtifact(a); err != nil {
		return err
	}
	if *apply && *expect != a.Manifest.DatasetVersion {
		return errors.New("-expect-version must match reviewed full manifest; .env version will not be changed")
	}
	configBytes, err := os.ReadFile(".env")
	if err != nil {
		return errors.New("cannot read backend/.env")
	}
	c, err := parseConfig(string(configBytes))
	if err != nil {
		return err
	}
	scopeSet := map[string]bool{}
	links := int64(a.Manifest.Counts["edges"])
	for _, r := range a.Records {
		for _, scope := range scopeIDs(r) {
			scopeSet[scope] = true
			links++
		}
	}
	newNodes := int64(len(a.Records) + len(a.Partitions) + len(a.SourcePartitions) + len(scopeSet) + 1)
	u, _ := url.Parse(c.URI)
	fmt.Printf("Target host=%s database=%s version=%s\nProjected namespace nodes=%d links=%d artifact_bytes=%d\n", u.Host, c.Database, a.Manifest.DatasetVersion, newNodes, links, len(data))
	if !*apply && !*inspect {
		fmt.Println("Dry run only; no network or writes.")
		return nil
	}
	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Minute)
	defer cancel()
	driver, err := neo4j.NewDriverWithContext(c.URI, neo4j.BasicAuth(c.Username, c.Password, ""))
	if err != nil {
		return safeError("driver", err)
	}
	defer driver.Close(context.Background())
	if err := driver.VerifyConnectivity(ctx); err != nil {
		return safeError("connection", err)
	}
	session := driver.NewSession(ctx, neo4j.SessionConfig{DatabaseName: c.Database, AccessMode: neo4j.AccessModeRead})
	defer session.Close(context.Background())
	res, err := session.Run(ctx, "MATCH (n) RETURN count(n) AS count", nil)
	if err != nil {
		return safeError("node count", err)
	}
	row, err := res.Single(ctx)
	if err != nil {
		return safeError("node count", err)
	}
	nodes, _ := row.AsMap()["count"].(int64)
	res, err = session.Run(ctx, "MATCH ()-[r]->() RETURN count(r) AS count", nil)
	if err != nil {
		return safeError("relationship count", err)
	}
	row, err = res.Single(ctx)
	if err != nil {
		return safeError("relationship count", err)
	}
	currentLinks, _ := row.AsMap()["count"].(int64)
	res, err = session.Run(ctx, "MATCH (m:RelioFullManifest {version:$version}) RETURN m.hash AS hash,m.ready AS ready", map[string]any{"version": a.Manifest.DatasetVersion})
	if err != nil {
		return safeError("manifest preflight", err)
	}
	existing, err := res.Collect(ctx)
	if err != nil {
		return safeError("manifest preflight", err)
	}
	if len(existing) > 1 {
		return errors.New("duplicate existing manifest")
	}
	additionalNodes, additionalLinks := newNodes, links
	if len(existing) == 1 {
		if existing[0].AsMap()["hash"] != a.Manifest.ManifestHash {
			return errors.New("namespace manifest differs; no overwrite allowed")
		}
		additionalNodes = 0
		additionalLinks = 0
	}
	if err := checkBudget(nodes, currentLinks, additionalNodes, additionalLinks, *nodeBudget, *linkBudget); err != nil {
		return err
	}
	fmt.Printf("Preflight existing nodes=%d links=%d; conservative capacity PASS\n", nodes, currentLinks)
	if !*apply {
		fmt.Println("Read-only preflight complete; no writes performed.")
		return nil
	}
	write := driver.NewSession(ctx, neo4j.SessionConfig{DatabaseName: c.Database, AccessMode: neo4j.AccessModeWrite})
	defer write.Close(context.Background())
	for _, query := range []string{
		"CREATE CONSTRAINT relio_full_manifest IF NOT EXISTS FOR (m:RelioFullManifest) REQUIRE m.version IS UNIQUE",
		"CREATE CONSTRAINT relio_full_record IF NOT EXISTS FOR (r:RelioFullRecord) REQUIRE (r.version,r.kind,r.id) IS UNIQUE",
		"CREATE CONSTRAINT relio_full_partition IF NOT EXISTS FOR (p:RelioFullPartition) REQUIRE (p.version,p.id) IS UNIQUE",
		"CREATE CONSTRAINT relio_full_source IF NOT EXISTS FOR (p:RelioFullSource) REQUIRE (p.version,p.id) IS UNIQUE",
		"CREATE CONSTRAINT relio_full_scope IF NOT EXISTS FOR (s:RelioFullScope) REQUIRE (s.version,s.scope) IS UNIQUE",
		"CREATE INDEX relio_full_time IF NOT EXISTS FOR (r:RelioFullRecord) ON (r.version,r.available_from,r.kind)",
	} {
		res, err := write.Run(ctx, query, nil)
		if err != nil {
			return safeError("schema", err)
		}
		if _, err := res.Consume(ctx); err != nil {
			return safeError("schema", err)
		}
	}
	exec := func(stage, query string, params map[string]any, want int64) error {
		_, err := write.ExecuteWrite(ctx, func(tx neo4j.ManagedTransaction) (any, error) {
			res, err := tx.Run(ctx, query, params)
			if err != nil {
				return nil, err
			}
			row, err := res.Single(ctx)
			if err != nil {
				return nil, err
			}
			got, _ := row.AsMap()["count"].(int64)
			if got != want {
				return nil, fmt.Errorf("immutable/hash/count check failed")
			}
			return nil, nil
		})
		if err != nil {
			return safeError(stage, err)
		}
		return nil
	}
	mJSON, _ := json.Marshal(a.Manifest)
	version := a.Manifest.DatasetVersion
	if err := exec("manifest", "MERGE (m:RelioFullManifest {version:$version}) ON CREATE SET m.hash=$hash,m.payload=$payload,m.ready=false WITH m WHERE m.hash=$hash AND m.payload=$payload RETURN count(m) AS count", map[string]any{"version": version, "hash": a.Manifest.ManifestHash, "payload": string(mJSON)}, 1); err != nil {
		return err
	}
	batch := 200
	for start := 0; start < len(a.Records); start += batch {
		end := start + batch
		if end > len(a.Records) {
			end = len(a.Records)
		}
		rows := []map[string]any{}
		for _, r := range a.Records[start:end] {
			blob, _ := json.Marshal(r)
			deal, account := "", ""
			if len(r.DealIDs) > 0 {
				deal = r.DealIDs[0]
			}
			if len(r.AccountIDs) > 0 {
				account = r.AccountIDs[0]
			}
			rows = append(rows, map[string]any{"kind": r.Kind, "id": r.ID, "envelope": string(blob), "hash": repository.FullRecordDigest(r), "available_from": string(r.AvailableFrom), "deal_id": deal, "account_id": account, "scopes": scopeIDs(r)})
		}
		query := "UNWIND $rows AS x MERGE (r:RelioFullRecord {version:$version,kind:x.kind,id:x.id}) ON CREATE SET r.hash=x.hash,r.envelope=x.envelope,r.available_from=x.available_from,r.deal_id=x.deal_id,r.account_id=x.account_id WITH r,x WHERE r.hash=x.hash AND r.envelope=x.envelope AND r.available_from=x.available_from AND r.deal_id=x.deal_id AND r.account_id=x.account_id FOREACH (scope IN x.scopes | MERGE (s:RelioFullScope {version:$version,scope:scope}) MERGE (s)-[:HAS_RECORD]->(r)) RETURN count(r) AS count"
		if err := exec("records", query, map[string]any{"version": version, "rows": rows}, int64(len(rows))); err != nil {
			return err
		}
		if start%2000 == 0 {
			fmt.Printf("Records loaded %d/%d\n", end, len(a.Records))
		}
	}
	for start := 0; start < len(a.Partitions); start += batch {
		end := start + batch
		if end > len(a.Partitions) {
			end = len(a.Partitions)
		}
		rows := []map[string]any{}
		for _, p := range a.Partitions[start:end] {
			blob, _ := json.Marshal(p)
			rows = append(rows, map[string]any{"id": p.PartitionID, "hash": p.Hash, "payload": string(blob)})
		}
		if err := exec("partitions", "UNWIND $rows AS x MERGE (p:RelioFullPartition {version:$version,id:x.id}) ON CREATE SET p.hash=x.hash,p.payload=x.payload WITH p,x WHERE p.hash=x.hash AND p.payload=x.payload RETURN count(p) AS count", map[string]any{"version": version, "rows": rows}, int64(len(rows))); err != nil {
			return err
		}
	}
	for start := 0; start < len(a.SourcePartitions); start += 20 {
		end := start + 20
		if end > len(a.SourcePartitions) {
			end = len(a.SourcePartitions)
		}
		rows := []map[string]any{}
		for _, p := range a.SourcePartitions[start:end] {
			blob, _ := json.Marshal(p)
			rows = append(rows, map[string]any{"id": p.ID, "hash": p.Hash, "payload": string(blob), "account_id": p.AccountID, "available_from": string(p.Date)})
		}
		if err := exec("source backing", "UNWIND $rows AS x MERGE (p:RelioFullSource {version:$version,id:x.id}) ON CREATE SET p.hash=x.hash,p.payload=x.payload,p.account_id=x.account_id,p.available_from=x.available_from WITH p,x WHERE p.hash=x.hash AND p.payload=x.payload AND p.account_id=x.account_id AND p.available_from=x.available_from RETURN count(p) AS count", map[string]any{"version": version, "rows": rows}, int64(len(rows))); err != nil {
			return err
		}
	}
	edges := []map[string]any{}
	for _, r := range a.Records {
		if r.Kind == "edges" {
			var e models.Edge
			_ = json.Unmarshal(r.Payload, &e)
			edges = append(edges, map[string]any{"id": e.EdgeID, "source": e.Source, "target": e.Target, "type": e.EdgeType})
		}
	}
	sort.Slice(edges, func(i, j int) bool { return edges[i]["id"].(string) < edges[j]["id"].(string) })
	for start := 0; start < len(edges); start += batch {
		end := start + batch
		if end > len(edges) {
			end = len(edges)
		}
		if err := exec("adjacency", "UNWIND $rows AS x MATCH (a:RelioFullRecord {version:$version,kind:'nodes',id:x.source}),(b:RelioFullRecord {version:$version,kind:'nodes',id:x.target}) MERGE (a)-[r:FULL_CONTEXT_LINK {version:$version,edge_id:x.id}]->(b) ON CREATE SET r.type=x.type WITH r,x WHERE r.type=x.type RETURN count(r) AS count", map[string]any{"version": version, "rows": edges[start:end]}, int64(end-start)); err != nil {
			return err
		}
	}
	// Every batch checked actual immutable values; final publication also checks exact coverage.
	query := "MATCH (m:RelioFullManifest {version:$version}) CALL { WITH m MATCH (r:RelioFullRecord {version:m.version}) RETURN count(r) AS records } CALL { WITH m MATCH (p:RelioFullPartition {version:m.version}) RETURN count(p) AS partitions } CALL { WITH m MATCH (p:RelioFullSource {version:m.version}) RETURN count(p) AS sources } CALL { WITH m MATCH ()-[r:FULL_CONTEXT_LINK {version:m.version}]->() RETURN count(r) AS links } WITH m,records,partitions,sources,links WHERE m.hash=$hash AND records=$records AND partitions=$partitions AND sources=$sources AND links=$links SET m.ready=true RETURN count(m) AS count"
	if err := exec("readiness", query, map[string]any{"version": version, "hash": a.Manifest.ManifestHash, "records": len(a.Records), "partitions": len(a.Partitions), "sources": len(a.SourcePartitions), "links": len(edges)}, 1); err != nil {
		return err
	}
	fmt.Printf("Full namespace ready: records=%d partitions=%d source_containers=%d links=%d\n", len(a.Records), len(a.Partitions), len(a.SourcePartitions), len(edges))
	return nil
}
func main() {
	if err := run(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}
