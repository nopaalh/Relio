package repository

const neo4jManifestQuery = `MATCH (d:RelioDataset {dataset_version:$version}) RETURN d.payload AS payload,d.ready AS ready,d.manifest_hash AS manifest_hash`

// P04 is a bounded immutable namespace. Read raw records privately and project
// every field/reference before returning domain data; never expose payloads.
const neo4jRecordsQuery = `MATCH (n:RelioRecord {dataset_version:$version}) RETURN n.record_kind AS kind,n.record_id AS id,n.payload AS payload,n.payload_hash AS payload_hash ORDER BY kind,id`
const neo4jLinksQuery = `MATCH (a:RelioRecord)-[r:CONTEXT_LINK {dataset_version:$version}]->(b:RelioRecord) RETURN a.record_id AS source,b.record_id AS target,r.edge_id AS edge_id,r.edge_type AS edge_type,a.record_kind AS source_kind,b.record_kind AS target_kind,a.dataset_version AS source_version,b.dataset_version AS target_version`
