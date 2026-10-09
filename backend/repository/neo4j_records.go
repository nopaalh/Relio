package repository

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"github.com/nopaalh/Relio/backend/models"
	"reflect"
	"sort"
	"strings"
)

const neo4jSchemaVersion = "relio-neo4j-p04-v1"

type storedManifest struct {
	DatasetVersion string               `json:"dataset_version"`
	SchemaVersion  string               `json:"schema_version"`
	ManifestHash   string               `json:"manifest_hash"`
	Ready          bool                 `json:"ready"`
	DealIDs        []string             `json:"deal_ids"`
	AccountIDs     []string             `json:"account_ids"`
	Sources        map[string]string    `json:"sources"`
	Counts         map[string]int       `json:"counts"`
	Records        []storedRecordDigest `json:"records"`
}

type storedRecordDigest struct {
	Kind string `json:"kind"`
	ID   string `json:"id"`
	Hash string `json:"hash"`
}

// Registry/coverage binding, not an authenticity signature or permission.
func manifestSeal(m storedManifest) string {
	lines := []string{"schema:" + m.SchemaVersion, "version:" + m.DatasetVersion}
	keys := []string{}
	for k := range m.Sources {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	for _, k := range keys {
		lines = append(lines, "source:"+k+":"+m.Sources[k])
	}
	keys = []string{}
	for k := range m.Counts {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	for _, k := range keys {
		lines = append(lines, fmt.Sprintf("count:%s:%d", k, m.Counts[k]))
	}
	records := append([]storedRecordDigest{}, m.Records...)
	sort.Slice(records, func(i, j int) bool {
		if records[i].Kind != records[j].Kind {
			return records[i].Kind < records[j].Kind
		}
		return records[i].ID < records[j].ID
	})
	for _, r := range records {
		lines = append(lines, "record:"+r.Kind+":"+r.ID+":"+r.Hash)
	}
	ids := append([]string{}, m.DealIDs...)
	sort.Strings(ids)
	for _, id := range ids {
		lines = append(lines, "deal:"+id)
	}
	ids = append([]string{}, m.AccountIDs...)
	sort.Strings(ids)
	for _, id := range ids {
		lines = append(lines, "account:"+id)
	}
	sum := sha256.Sum256([]byte(strings.Join(lines, "\n")))
	return hex.EncodeToString(sum[:])
}

type storedDeal struct {
	Value         models.DealFacts `json:"value"`
	AvailableFrom models.Date      `json:"available_from"`
}
type storedNode struct {
	Value         models.Node `json:"value"`
	AvailableFrom models.Date `json:"available_from"`
}
type storedEdge struct {
	Value         models.Edge `json:"value"`
	AvailableFrom models.Date `json:"available_from"`
}
type storedEvidence struct {
	Value         models.Evidence `json:"value"`
	AvailableFrom models.Date     `json:"available_from"`
}
type storedContext struct {
	Manifest storedManifest   `json:"manifest"`
	Deals    []storedDeal     `json:"deals"`
	Nodes    []storedNode     `json:"nodes"`
	Edges    []storedEdge     `json:"edges"`
	Events   []models.Event   `json:"events"`
	Evidence []storedEvidence `json:"evidence"`
}

// Validate the immutable namespace, including nested facts and references,
// before any projection. A corrupt namespace must not look like empty data.
func validateStoredContext(raw storedContext) error {
	if raw.Manifest.ManifestHash != manifestSeal(raw.Manifest) || len(raw.Manifest.DealIDs) != 1 || raw.Manifest.DealIDs[0] != "DL-004" || len(raw.Manifest.AccountIDs) != 1 || raw.Manifest.AccountIDs[0] != "P04" || len(raw.Deals) != 1 || raw.Deals[0].Value.DealID != "DL-004" || raw.Deals[0].Value.AccountID != "P04" {
		return fmt.Errorf("invalid sealed coverage/native anchor")
	}
	counts := map[string]int{"deals": len(raw.Deals), "nodes": len(raw.Nodes), "edges": len(raw.Edges), "events": len(raw.Events), "evidence": len(raw.Evidence)}
	for k, v := range counts {
		if raw.Manifest.Counts[k] != v {
			return fmt.Errorf("invalid %s count", k)
		}
	}
	nodes, edges, events, evidence := map[string]bool{}, map[string]bool{}, map[string]bool{}, map[string]bool{}
	add := func(m map[string]bool, id string) error {
		if id == "" || m[id] {
			return fmt.Errorf("duplicate/empty ID")
		}
		m[id] = true
		return nil
	}
	for _, n := range raw.Nodes {
		if err := add(nodes, n.Value.NodeID); err != nil {
			return err
		}
	}
	for _, e := range raw.Edges {
		if err := add(edges, e.Value.EdgeID); err != nil {
			return err
		}
		if !nodes[e.Value.Source] || !nodes[e.Value.Target] {
			return fmt.Errorf("missing endpoint")
		}
	}
	for _, e := range raw.Events {
		if err := add(events, e.EventID); err != nil {
			return err
		}
	}
	for _, e := range raw.Evidence {
		if err := add(evidence, e.Value.EvidenceID); err != nil {
			return err
		}
	}
	anchor := false
	for _, e := range raw.Edges {
		if e.Value.EdgeType == "ACCOUNT_CONTEXT" && e.Value.Source == "deal:DL-004" && e.Value.Target == "account:P04" {
			anchor = true
		}
	}
	if !anchor {
		return fmt.Errorf("native account anchor missing")
	}
	sets := map[string]map[string]bool{"EvidenceIDs": evidence, "RecordEvidenceIDs": evidence, "EventIDs": events, "EdgeIDs": edges, "NodeIDs": nodes, "CandidateNodeIDs": nodes}
	var walk func(reflect.Value) error
	walk = func(v reflect.Value) error {
		if v.Kind() == reflect.Pointer {
			if !v.IsNil() {
				return walk(v.Elem())
			}
			return nil
		}
		if v.CanInterface() {
			if f, ok := v.Interface().(interface{ Validate() error }); ok {
				if err := f.Validate(); err != nil {
					return err
				}
			}
			if p, ok := v.Interface().(models.ParticipantRef); ok && p.NodeID != nil && !nodes[*p.NodeID] {
				return fmt.Errorf("missing participant")
			}
			if d, ok := v.Interface().(models.Date); ok {
				if _, err := models.ParseDate(string(d)); err != nil {
					return err
				}
			}
		}
		switch v.Kind() {
		case reflect.Struct:
			for i := 0; i < v.NumField(); i++ {
				f := v.Field(i)
				if set, ok := sets[v.Type().Field(i).Name]; ok {
					for j := 0; j < f.Len(); j++ {
						if !set[f.Index(j).String()] {
							return fmt.Errorf("missing nested reference")
						}
					}
				}
				if err := walk(f); err != nil {
					return err
				}
			}
		case reflect.Slice:
			if v.IsNil() {
				return fmt.Errorf("nil collection")
			}
			for i := 0; i < v.Len(); i++ {
				if err := walk(v.Index(i)); err != nil {
					return err
				}
			}
		case reflect.Map:
			if v.IsNil() {
				return fmt.Errorf("nil map")
			}
		}
		return nil
	}
	return walk(reflect.ValueOf(raw))
}
