CREATE TABLE IF NOT EXISTS visits_find_peers (
    visit_round UInt64,
    timestamp DateTime,
    key String,
    blob_number UInt64,
    duration_ms Int64,
    items Int32,
    peers UInt32,
    error String
) ENGINE = ReplicatedMergeTree () PRIMARY KEY (visit_round) TTL toDateTime (timestamp) + INTERVAL 180 DAY;
