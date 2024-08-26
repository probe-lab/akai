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
    block_number UInt64,
    rows UInt32,
    columns UInt32,
    tracking_ttl DateTime
)
ENGINE = MergeTree()
PRIMARY KEY (hash, block_number);


CREATE TABLE IF NOT EXISTS segments (
    timestamp DateTime,
    block_number UInt64,
    key String,
    row UInt32,
    column UInt32,
    tracking_tll DateTime
)
ENGINE = MergeTree()
PRIMARY KEY (block_number, key);


CREATE TABLE IF NOT EXISTS visits (
    timestamp DateTime,
    key String,
    duration_ms Int64,
    is_retrievable Bool
)
ENGINE = MergeTree()
PRIMARY KEY (key);