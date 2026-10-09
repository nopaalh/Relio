CREATE CONSTRAINT relio_dataset IF NOT EXISTS FOR (d:RelioDataset) REQUIRE d.dataset_version IS UNIQUE;
CREATE CONSTRAINT relio_record IF NOT EXISTS FOR (n:RelioRecord) REQUIRE (n.dataset_version,n.record_kind,n.record_id) IS UNIQUE;
