package repository

import (
	"context"
	"encoding/json"
	"errors"
	"github.com/neo4j/neo4j-go-driver/v5/neo4j"
	"net/url"
	"sort"
	"strings"
	"sync"
)

type Neo4jFullContextRepository struct {
	read  func(context.Context, fullQuery) (fullSelection, error)
	close func(context.Context) error
	// O2 may configure this after measured budget agreement; zero means no imposed cap.
	MaxSelectedEvidenceBytes int
}

func (r *Neo4jFullContextRepository) Close(ctx context.Context) error {
	if r == nil || r.close == nil {
		return nil
	}
	return r.close(ctx)
}
func NewNeo4jFullContextRepository(ctx context.Context, c Neo4jConfig) (*Neo4jFullContextRepository, error) {
	u, err := url.Parse(c.URI)
	if err != nil || u.Scheme != "neo4j+s" || u.Host == "" || u.User != nil || u.RawQuery != "" || u.Fragment != "" || strings.TrimSpace(c.Username) == "" || c.Password == "" || strings.TrimSpace(c.Database) == "" {
		return nil, &RepositoryError{Code: DataNotReady}
	}
	driver, err := neo4j.NewDriverWithContext(c.URI, neo4j.BasicAuth(c.Username, c.Password, ""))
	if err != nil {
		return nil, &RepositoryError{Code: DataNotReady, Cause: err}
	}
	if err := driver.VerifyConnectivity(ctx); err != nil {
		_ = driver.Close(ctx)
		return nil, &RepositoryError{Code: DataNotReady, Cause: err}
	}
	r := &Neo4jFullContextRepository{close: driver.Close}
	// Immutable manifest roots cached per dataset version; readiness/hash is checked
	// on every read. Business records and model results are never cached here.
	manifests := map[string]FullManifest{}
	var mutex sync.Mutex
	r.read = func(ctx context.Context, q fullQuery) (fullSelection, error) {
		session := driver.NewSession(ctx, neo4j.SessionConfig{DatabaseName: c.Database, AccessMode: neo4j.AccessModeRead})
		defer session.Close(context.Background())
		data, err := session.ExecuteRead(ctx, func(tx neo4j.ManagedTransaction) (any, error) {
			header, err := tx.Run(ctx, fullManifestHeaderQuery, map[string]any{"version": q.Version})
			if err != nil {
				return nil, err
			}
			rows, err := header.Collect(ctx)
			if err != nil {
				return nil, err
			}
			if len(rows) != 1 || rows[0].AsMap()["ready"] != true {
				return nil, &RepositoryError{Code: DataNotReady}
			}
			hash, _ := rows[0].AsMap()["hash"].(string)
			mutex.Lock()
			m, cached := manifests[q.Version]
			mutex.Unlock()
			if !cached {
				res, err := tx.Run(ctx, fullManifestPayloadQuery, map[string]any{"version": q.Version})
				if err != nil {
					return nil, err
				}
				row, err := res.Single(ctx)
				if err != nil {
					return nil, err
				}
				payload, _ := row.AsMap()["payload"].(string)
				if json.Unmarshal([]byte(payload), &m) != nil || m.SchemaVersion != FullSchemaVersion || m.DatasetVersion != q.Version || m.ManifestHash != FullManifestDigest(m) {
					return nil, &RepositoryError{Code: DataNotReady}
				}
				mutex.Lock()
				manifests[q.Version] = m
				mutex.Unlock()
			}
			if m.ManifestHash != hash {
				return nil, &RepositoryError{Code: DataNotReady}
			}
			scopes := append([]string{}, q.Accounts...)
			if q.Company {
				scopes = append(scopes, "company")
			}
			lookupKind := q.LookupKind
			if lookupKind == "" {
				lookupKind = "evidence"
			}
			params := map[string]any{"version": q.Version, "scopes": scopes, "kinds": q.Kinds, "as_of": string(q.AsOf), "deals": q.Deals, "analogs": q.Analogs, "lookup_id": q.LookupID, "lookup_kind": lookupKind}
			res, err := tx.Run(ctx, fullRecordsQuery, params)
			if err != nil {
				return nil, err
			}
			rows, err = res.Collect(ctx)
			if err != nil {
				return nil, err
			}
			out := fullSelection{Manifest: m, Records: []FullRecord{}, Partitions: []FullPartition{}}
			partIDs := map[string]bool{}
			for _, row := range rows {
				blob, _ := row.AsMap()["envelope"].(string)
				var record FullRecord
				if json.Unmarshal([]byte(blob), &record) != nil {
					return nil, &RepositoryError{Code: DataNotReady}
				}
				out.Records = append(out.Records, record)
				partIDs[record.PartitionID] = true
			}
			ids := []string{}
			for id := range partIDs {
				ids = append(ids, id)
			}
			sort.Strings(ids)
			res, err = tx.Run(ctx, fullPartitionsQuery, map[string]any{"version": q.Version, "ids": ids})
			if err != nil {
				return nil, err
			}
			rows, err = res.Collect(ctx)
			if err != nil {
				return nil, err
			}
			for _, row := range rows {
				blob, _ := row.AsMap()["payload"].(string)
				var p FullPartition
				if json.Unmarshal([]byte(blob), &p) != nil {
					return nil, &RepositoryError{Code: DataNotReady}
				}
				out.Partitions = append(out.Partitions, p)
			}
			if err := verifyFullSelection(m, out.Partitions, out.Records); err != nil {
				return nil, err
			}
			if strings.HasPrefix(q.LookupID, "ev:usage:") {
				sourceID := "usage-source:" + strings.TrimPrefix(q.LookupID, "ev:usage:")
				res, e := tx.Run(ctx, fullSourceQuery, map[string]any{"version": q.Version, "id": sourceID, "accounts": q.Accounts, "as_of": string(q.AsOf)})
				if e != nil {
					return nil, e
				}
				rows, e := res.Collect(ctx)
				if e != nil {
					return nil, e
				}
				for _, row := range rows {
					blob, _ := row.AsMap()["payload"].(string)
					var p FullSourcePartition
					if json.Unmarshal([]byte(blob), &p) != nil || p.ID != sourceID || p.Hash != FullSourceDigest(p) || m.PartitionRoots["source:"+p.ID] != p.Hash {
						return nil, &RepositoryError{Code: DataNotReady}
					}
					out.SourcePartitions = append(out.SourcePartitions, p)
				}
			}
			return out, nil
		})
		if err != nil {
			var re *RepositoryError
			if errors.As(err, &re) {
				return fullSelection{}, err
			}
			return fullSelection{}, &RepositoryError{Code: QueryFailed, Cause: err}
		}
		return data.(fullSelection), nil
	}
	return r, nil
}
