CREATE TABLE IF NOT EXISTS networks (
    network_id UInt16,
    protocol String,
    network_name String
)
ENGINE = MergeTree()
PRIMARY KEY (network_id);


CREATE TABLE IF NOT EXISTS blobs (
    network_id UInt16,
    timestamp DateTime,
    hash String,
    key String,
    blob_number UInt64,
    rows UInt32,
    columns UInt32,
    sample_until DateTime
)
ENGINE = MergeTree()
PRIMARY KEY (hash, blob_number);


CREATE TABLE IF NOT EXISTS segments (
    timestamp DateTime,
    blob_number UInt64,
    key String,
    row UInt32,
    column UInt32,
    sample_until DateTime
)
ENGINE = MergeTree()
PRIMARY KEY (blob_number, key);


CREATE TABLE IF NOT EXISTS visits (
    timestamp DateTime,
    key String,
    blob_number UInt64,
    row UInt32,
    column UInt32,
    duration_ms Int64,
    is_retrievable Bool,
    providers UInt32,
    bytes UInt32,
    error String
)
ENGINE = MergeTree()
PRIMARY KEY (key);