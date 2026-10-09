package repository

import (
	"encoding/json"
	"fmt"
)

func verifyFullSelection(m FullManifest, parts []FullPartition, records []FullRecord) error {
	invalid := func() error { return &RepositoryError{Code: DataNotReady} }
	if m.SchemaVersion != FullSchemaVersion || m.ManifestHash != FullManifestDigest(m) {
		return invalid()
	}
	registry := map[string]map[string]string{}
	for _, p := range parts {
		if p.Hash != FullPartitionDigest(p) || m.PartitionRoots[p.PartitionID] != p.Hash || registry[p.PartitionID] != nil {
			return invalid()
		}
		ids := map[string]string{}
		for _, d := range p.Digests {
			key := d.Kind + "|" + d.ID
			if ids[key] != "" {
				return invalid()
			}
			ids[key] = d.Hash
		}
		registry[p.PartitionID] = ids
	}
	seen := map[string]bool{}
	idFields := map[string]string{"nodes": "node_id", "edges": "edge_id", "deals": "deal_id", "events": "event_id", "evidence": "evidence_id", "templates": "action_id", "occurrences": "occurrence_id"}
	for _, r := range records {
		key := r.Kind + "|" + r.ID
		if seen[key] || r.PayloadHash != fullHash(r.Payload) || registry[r.PartitionID][key] != FullRecordDigest(r) {
			return invalid()
		}
		seen[key] = true
		var body map[string]json.RawMessage
		var id string
		if json.Unmarshal(r.Payload, &body) != nil || json.Unmarshal(body[idFields[r.Kind]], &id) != nil || id != r.ID {
			return &RepositoryError{Code: DataNotReady, Cause: fmt.Errorf("record native ID mismatch")}
		}
	}
	return nil
}
