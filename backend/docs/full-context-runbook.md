# Full static context graph runbook

Run commands from backend/. Go and existing Neo4j driver are required. Dataset is FIX; preparation is one-off, not ETL, scheduler or runtime CSV ingestion. Keep backend/.env and generated full/artifact.json out of Git.

1. Prepare/validate all15 sources:

```powershell
go run ./tools/preparefull -dataset ../../Datasets -out database/neo4j/full
go run ./tools/loadfull
go run ./tools/loadfull -inspect
```

The loader reads local .env safely (NEO4J_URI, NEO4J_USERNAME, NEO4J_PASSWORD, NEO4J_DATABASE); it never evaluates shell code or prints credentials. Use the actual Aura database name, not an assumed neo4j. Inspect is read-only and reports existing/projected counts. Conservative engineering ceilings50k nodes/175k relationships are configurable, not an assertion about subscription tier.

2. Explicitly load the prepared immutable version:

```powershell
$manifest = Get-Content database/neo4j/full/manifest.json -Raw | ConvertFrom-Json
go run ./tools/loadfull -apply -expect-version $manifest.dataset_version
```

Constraints, batch MERGE, source backing and graph adjacency are automatic: no statement-by-statement manual paste. Existing P04 namespace is preserved. No DELETE/overwrite, paid upgrade or automatic .env version switch. A failed first load remains unready; rerun the same sealed artifact/version. Changed mapping/content must use another version. Ready publishes only after checked immutable values and exact coverage. Rerun should not add duplicate records.

3. Local prepared acceptance:

```powershell
$env:RELIO_FULL_ARTIFACT_TEST = (Resolve-Path database/neo4j/full/artifact.json).Path
go test ./repository -run TestFullPreparedAcceptance -count=1 -v
```

4. Live read-only acceptance after publishing: export Neo4j credentials from your local configuration into process environment without logging them. Set RELIO_FULL_INTEGRATION=1 and RELIO_FULL_DATASET_VERSION to the manifest version, then:

```powershell
go test ./repository -run TestNeo4jFullLive -count=1 -v
```

Full and P04 integration flags/versions are separate. Default suite skips opt-in tests; enabled missing configuration fails. Tests never import seed or modify Aura. Normal checks: go test ./... -count=1; go vet ./...; go test -race ./... -count=1. Counts/measurements and actual live status are recorded in full-context-o2-handoff.md after execution.
