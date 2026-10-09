package repository

import (
	"context"
	"encoding/json"
	"github.com/neo4j/neo4j-go-driver/v5/neo4j"
	"github.com/nopaalh/Relio/backend/models"
	"net/url"
	"strings"
)

type Neo4jConfig struct {
	URI      string
	Username string
	Password string
	Database string
}

func NewNeo4jContextRepository(ctx context.Context, c Neo4jConfig) (*Neo4jContextRepository, error) {
	u, err := url.Parse(c.URI)
	if err != nil || u.Scheme != "neo4j+s" || u.Host == "" || u.User != nil || u.RawQuery != "" || strings.TrimSpace(c.Username) == "" || c.Password == "" || strings.TrimSpace(c.Database) == "" {
		return nil, &RepositoryError{Code: DataNotReady}
	}
	driver, err := neo4j.NewDriverWithContext(c.URI, neo4j.BasicAuth(c.Username, c.Password, ""))
	if err != nil {
		return nil, &RepositoryError{Code: DataNotReady, Cause: err}
	}
	if err = driver.VerifyConnectivity(ctx); err != nil {
		_ = driver.Close(context.Background())
		return nil, &RepositoryError{Code: DataNotReady, Cause: err}
	}
	r := &Neo4jContextRepository{close: driver.Close}
	r.read = func(ctx context.Context, s models.SnapshotContext) (storedContext, error) {
		session := driver.NewSession(ctx, neo4j.SessionConfig{DatabaseName: c.Database, AccessMode: neo4j.AccessModeRead})
		defer session.Close(context.Background())
		data, err := session.ExecuteRead(ctx, func(tx neo4j.ManagedTransaction) (any, error) {
			params := map[string]any{"version": s.DatasetVersion}
			result, err := tx.Run(ctx, neo4jManifestQuery, params)
			if err != nil {
				return nil, err
			}
			records, err := result.Collect(ctx)
			if err != nil {
				return nil, err
			}
			if len(records) != 1 {
				return nil, &RepositoryError{Code: DataNotReady}
			}
			raw := storedContext{Deals: []storedDeal{}, Nodes: []storedNode{}, Edges: []storedEdge{}, Events: []models.Event{}, Evidence: []storedEvidence{}}
			row := records[0].AsMap()
			payload, ok := row["payload"].(string)
			if !ok {
				return nil, &RepositoryError{Code: DataNotReady}
			}
			if err = json.Unmarshal([]byte(payload), &raw.Manifest); err != nil {
				return nil, &RepositoryError{Code: DataNotReady, Cause: err}
			}
			raw.Manifest.Ready, _ = row["ready"].(bool)
			if !raw.Manifest.Ready || row["manifest_hash"] != raw.Manifest.ManifestHash {
				return nil, &RepositoryError{Code: DataNotReady}
			}
			result, err = tx.Run(ctx, neo4jRecordsQuery, params)
			if err != nil {
				return nil, err
			}
			records, err = result.Collect(ctx)
			if err != nil {
				return nil, err
			}
			rows := []neo4jRecordRow{}
			for _, record := range records {
				fields := record.AsMap()
				kind, _ := fields["kind"].(string)
				id, _ := fields["id"].(string)
				payload, _ := fields["payload"].(string)
				hash, _ := fields["payload_hash"].(string)
				rows = append(rows, neo4jRecordRow{kind, id, payload, hash})
			}
			raw, err = decodeNeo4jRecords(raw.Manifest, rows)
			if err != nil {
				return nil, err
			}
			result, err = tx.Run(ctx, neo4jLinksQuery, params)
			if err != nil {
				return nil, err
			}
			links, err := result.Collect(ctx)
			if err != nil {
				return nil, err
			}
			if len(links) != len(raw.Edges) {
				return nil, &RepositoryError{Code: DataNotReady}
			}
			edges := map[string]models.Edge{}
			for _, e := range raw.Edges {
				edges[e.Value.EdgeID] = e.Value
			}
			seen := map[string]bool{}
			for _, link := range links {
				row = link.AsMap()
				id, _ := row["edge_id"].(string)
				e, ok := edges[id]
				if !ok || seen[id] || row["source"] != e.Source || row["target"] != e.Target || row["edge_type"] != e.EdgeType || row["source_kind"] != "nodes" || row["target_kind"] != "nodes" || row["source_version"] != s.DatasetVersion || row["target_version"] != s.DatasetVersion {
					return nil, &RepositoryError{Code: DataNotReady}
				}
				seen[id] = true
			}
			if err = validateStoredContext(raw); err != nil {
				return nil, &RepositoryError{Code: DataNotReady, Cause: err}
			}
			return raw, nil
		})
		if err != nil {
			return storedContext{}, err
		}
		return data.(storedContext), nil
	}
	return r, nil
}
