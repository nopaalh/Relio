package repository

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"github.com/nopaalh/Relio/backend/models"
	"sort"
)

const FullSchemaVersion = "relio-neo4j-full-v1"

type FullManifest struct {
	DealAccountIDs map[string]string      `json:"deal_account_ids"`
	DealCreatedAt  map[string]models.Date `json:"deal_created_at"`
	SchemaVersion  string                 `json:"schema_version"`
	DatasetVersion string                 `json:"dataset_version"`
	ManifestHash   string                 `json:"manifest_hash"`
	Sources        map[string]string      `json:"sources"`
	SourceCounts   map[string]int         `json:"source_counts"`
	Counts         map[string]int         `json:"counts"`
	DealIDs        []string               `json:"deal_ids"`
	AccountIDs     []string               `json:"account_ids"`
	PartitionRoots map[string]string      `json:"partition_roots"`
}
type FullRecord struct {
	Kind          string          `json:"kind"`
	ID            string          `json:"id"`
	PartitionID   string          `json:"partition_id"`
	AccountIDs    []string        `json:"account_ids"`
	DealIDs       []string        `json:"deal_ids"`
	AvailableFrom models.Date     `json:"available_from"`
	Payload       json.RawMessage `json:"payload"`
	PayloadHash   string          `json:"payload_hash"`
}
type FullDigest struct {
	Kind string `json:"kind"`
	ID   string `json:"id"`
	Hash string `json:"hash"`
}
type FullPartition struct {
	PartitionID string       `json:"partition_id"`
	Hash        string       `json:"hash"`
	Digests     []FullDigest `json:"digests"`
}
type FullSourcePartition struct {
	ID             string              `json:"id"`
	SourceFile     string              `json:"source_file"`
	SourceChecksum string              `json:"source_checksum"`
	AccountID      string              `json:"account_id"`
	Date           models.Date         `json:"date"`
	Rows           []map[string]string `json:"rows"`
	Keys           []map[string]string `json:"keys"`
	Hash           string              `json:"hash"`
}

// Offline tooling types, not HTTP DTOs or service-side database driver types.
type FullArtifact struct {
	Manifest         FullManifest          `json:"manifest"`
	Partitions       []FullPartition       `json:"partitions"`
	Records          []FullRecord          `json:"records"`
	SourcePartitions []FullSourcePartition `json:"source_partitions"`
}

func fullHash(b []byte) string                      { s := sha256.Sum256(b); return hex.EncodeToString(s[:]) }
func FullDigestJSON(v any) string                   { b, _ := json.Marshal(v); return fullHash(b) }
func FullRecordDigest(r FullRecord) string          { return FullDigestJSON(r) }
func FullPartitionDigest(p FullPartition) string    { p.Hash = ""; return FullDigestJSON(p) }
func FullManifestDigest(m FullManifest) string      { m.ManifestHash = ""; return FullDigestJSON(m) }
func FullSourceDigest(p FullSourcePartition) string { p.Hash = ""; return FullDigestJSON(p) }
func SealFullArtifact(a *FullArtifact) error {
	a.Manifest.SchemaVersion = FullSchemaVersion
	a.Manifest.PartitionRoots = map[string]string{}
	a.Manifest.Counts = map[string]int{}
	a.Partitions = []FullPartition{}
	sort.Slice(a.Records, func(i, j int) bool {
		if a.Records[i].Kind != a.Records[j].Kind {
			return a.Records[i].Kind < a.Records[j].Kind
		}
		return a.Records[i].ID < a.Records[j].ID
	})
	groups := map[string][]FullDigest{}
	for i := range a.Records {
		r := &a.Records[i]
		r.PayloadHash = fullHash(r.Payload)
		sort.Strings(r.AccountIDs)
		sort.Strings(r.DealIDs)
		groups[r.PartitionID] = append(groups[r.PartitionID], FullDigest{r.Kind, r.ID, FullRecordDigest(*r)})
		a.Manifest.Counts[r.Kind]++
	}
	keys := []string{}
	for k := range groups {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	for _, k := range keys {
		p := FullPartition{PartitionID: k, Digests: groups[k]}
		p.Hash = FullPartitionDigest(p)
		a.Partitions = append(a.Partitions, p)
		a.Manifest.PartitionRoots[k] = p.Hash
	}
	sort.Slice(a.SourcePartitions, func(i, j int) bool { return a.SourcePartitions[i].ID < a.SourcePartitions[j].ID })
	for i := range a.SourcePartitions {
		p := &a.SourcePartitions[i]
		p.Hash = FullSourceDigest(*p)
		a.Manifest.PartitionRoots["source:"+p.ID] = p.Hash
	}
	sort.Strings(a.Manifest.DealIDs)
	sort.Strings(a.Manifest.AccountIDs)
	a.Manifest.ManifestHash = FullManifestDigest(a.Manifest)
	return ValidateFullArtifact(*a)
}
func ValidateFullArtifact(a FullArtifact) error {
	if a.Manifest.SchemaVersion != FullSchemaVersion || a.Manifest.DatasetVersion == "" || a.Manifest.ManifestHash != FullManifestDigest(a.Manifest) {
		return fmt.Errorf("invalid full manifest seal")
	}
	registry := map[string]string{}
	parts := map[string]bool{}
	for _, p := range a.Partitions {
		if parts[p.PartitionID] || p.Hash != FullPartitionDigest(p) || a.Manifest.PartitionRoots[p.PartitionID] != p.Hash {
			return fmt.Errorf("invalid partition seal")
		}
		parts[p.PartitionID] = true
		for _, d := range p.Digests {
			key := d.Kind + "|" + d.ID
			if registry[key] != "" {
				return fmt.Errorf("duplicate record digest")
			}
			registry[key] = d.Hash
		}
	}
	seen := map[string]bool{}
	counts := map[string]int{}
	ids := map[string]string{"deals": "deal_id", "nodes": "node_id", "edges": "edge_id", "events": "event_id", "evidence": "evidence_id", "templates": "action_id", "occurrences": "occurrence_id"}
	for _, r := range a.Records {
		key := r.Kind + "|" + r.ID
		if r.ID == "" || seen[key] || !parts[r.PartitionID] || r.PayloadHash != fullHash(r.Payload) || registry[key] != FullRecordDigest(r) {
			return fmt.Errorf("invalid record binding")
		}
		seen[key] = true
		counts[r.Kind]++
		if _, err := models.ParseDate(string(r.AvailableFrom)); err != nil || r.AvailableFrom > MaxAsOf {
			return fmt.Errorf("invalid availability date")
		}
		var body map[string]json.RawMessage
		if json.Unmarshal(r.Payload, &body) != nil || ids[r.Kind] == "" {
			return fmt.Errorf("invalid record payload kind")
		}
		var id string
		if json.Unmarshal(body[ids[r.Kind]], &id) != nil || id != r.ID {
			return fmt.Errorf("native payload ID differs")
		}
	}
	if len(seen) != len(registry) {
		return fmt.Errorf("missing records")
	}
	if FullDigestJSON(counts) != FullDigestJSON(a.Manifest.Counts) {
		return fmt.Errorf("record counts differ")
	}
	for _, p := range a.SourcePartitions {
		k := "source:" + p.ID
		if parts[k] || p.Hash != FullSourceDigest(p) || a.Manifest.PartitionRoots[k] != p.Hash || len(p.Rows) != len(p.Keys) {
			return fmt.Errorf("invalid source partition")
		}
		parts[k] = true
	}
	if len(parts) != len(a.Manifest.PartitionRoots) {
		return fmt.Errorf("missing partition")
	}
	return nil
}
