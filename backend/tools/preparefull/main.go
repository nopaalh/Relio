package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"github.com/nopaalh/Relio/backend/models"
	"github.com/nopaalh/Relio/backend/repository"
	"os"
	"path/filepath"
)

func validateGraph(a repository.FullArtifact) error {
	nodes, edges, events, evidence := map[string]bool{}, map[string]bool{}, map[string]bool{}, map[string]bool{}
	for _, r := range a.Records {
		switch r.Kind {
		case "nodes":
			nodes[r.ID] = true
		case "edges":
			edges[r.ID] = true
		case "events":
			events[r.ID] = true
		case "evidence":
			evidence[r.ID] = true
		}
	}
	for _, r := range a.Records {
		if r.Kind == "edges" {
			var e models.Edge
			if json.Unmarshal(r.Payload, &e) != nil || !nodes[e.Source] || !nodes[e.Target] {
				return fmt.Errorf("dangling edge %s", r.ID)
			}
			for _, id := range e.EvidenceIDs {
				if !evidence[id] {
					return fmt.Errorf("edge proof absent")
				}
			}
		}
		if r.Kind == "events" {
			var e models.Event
			_ = json.Unmarshal(r.Payload, &e)
			for _, id := range e.NodeIDs {
				if !nodes[id] {
					return fmt.Errorf("event node absent %s", id)
				}
			}
			for _, id := range e.EvidenceIDs {
				if !evidence[id] {
					return fmt.Errorf("event proof absent")
				}
			}
		}
	}
	return nil
}
func run() error {
	dir := flag.String("dataset", "../../Datasets", "fixed source dataset")
	out := flag.String("out", "database/neo4j/full", "generated output directory")
	flag.Parse()
	srcAbs, _ := filepath.Abs(*dir)
	outAbs, _ := filepath.Abs(*out)
	rel, err := filepath.Rel(srcAbs, outAbs)
	if err != nil || rel == "." || (len(rel) < 2 || rel[:2] != "..") {
		return fmt.Errorf("output must not overwrite dataset")
	}
	s, err := readSources(*dir, "docs/source-audit-snapshot.json")
	if err != nil {
		return err
	}
	a, err := build(s)
	if err != nil {
		return err
	}
	data, err := json.Marshal(a)
	if err != nil {
		return err
	}
	if err := os.MkdirAll(*out, 0755); err != nil {
		return err
	}
	if err := os.WriteFile(filepath.Join(*out, "artifact.json"), append(data, '\n'), 0600); err != nil {
		return err
	}
	manifest, _ := json.MarshalIndent(a.Manifest, "", "  ")
	if err := os.WriteFile(filepath.Join(*out, "manifest.json"), append(manifest, '\n'), 0600); err != nil {
		return err
	}
	count := 0
	for _, n := range s.counts {
		count += n
	}
	fmt.Printf("Prepared %s\nSources=%d rows=%d records=%d partitions=%d backing_month_containers=%d bytes=%d\n", a.Manifest.DatasetVersion, len(s.counts), count, len(a.Records), len(a.Partitions), len(a.SourcePartitions), len(data))
	fmt.Printf("Domain counts: %v\n", a.Manifest.Counts)
	return nil
}
func main() {
	if err := run(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}
