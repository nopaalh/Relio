package repository

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"github.com/nopaalh/Relio/backend/models"
)

type neo4jRecordRow struct {
	Kind    string
	ID      string
	Payload string
	Hash    string
}

func decodeNeo4jRecords(m storedManifest, rows []neo4jRecordRow) (storedContext, error) {
	raw := storedContext{Manifest: m, Deals: []storedDeal{}, Nodes: []storedNode{}, Edges: []storedEdge{}, Events: []models.Event{}, Evidence: []storedEvidence{}}
	invalid := func(err error) (storedContext, error) {
		return storedContext{}, &RepositoryError{Code: DataNotReady, Cause: err}
	}
	if m.ManifestHash != manifestSeal(m) || len(rows) != len(m.Records) {
		return invalid(fmt.Errorf("manifest seal/count mismatch"))
	}
	registry := map[string]string{}
	for _, record := range m.Records {
		key := record.Kind + "|" + record.ID
		if registry[key] != "" || len(record.Hash) != 64 {
			return invalid(fmt.Errorf("invalid manifest record registry"))
		}
		registry[key] = record.Hash
	}
	seen := map[string]bool{}
	for _, row := range rows {
		key := row.Kind + "|" + row.ID
		sum := sha256.Sum256([]byte(row.Payload))
		hash := hex.EncodeToString(sum[:])
		if seen[key] || row.Hash != hash || registry[key] != hash {
			return invalid(fmt.Errorf("payload not bound to manifest"))
		}
		seen[key] = true
		var err error
		id := ""
		switch row.Kind {
		case "deals":
			var v storedDeal
			err = json.Unmarshal([]byte(row.Payload), &v)
			id = v.Value.DealID
			raw.Deals = append(raw.Deals, v)
		case "nodes":
			var v storedNode
			err = json.Unmarshal([]byte(row.Payload), &v)
			id = v.Value.NodeID
			raw.Nodes = append(raw.Nodes, v)
		case "edges":
			var v storedEdge
			err = json.Unmarshal([]byte(row.Payload), &v)
			id = v.Value.EdgeID
			raw.Edges = append(raw.Edges, v)
		case "events":
			var v models.Event
			err = json.Unmarshal([]byte(row.Payload), &v)
			id = v.EventID
			raw.Events = append(raw.Events, v)
		case "evidence":
			var v storedEvidence
			err = json.Unmarshal([]byte(row.Payload), &v)
			id = v.Value.EvidenceID
			raw.Evidence = append(raw.Evidence, v)
		default:
			err = fmt.Errorf("unknown record kind")
		}
		if err != nil {
			return invalid(err)
		}
		if id != row.ID {
			return invalid(fmt.Errorf("record ID mismatch"))
		}
	}
	if err := validateStoredContext(raw); err != nil {
		return invalid(err)
	}
	return raw, nil
}
