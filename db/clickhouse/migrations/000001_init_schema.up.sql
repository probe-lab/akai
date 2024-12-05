CREATE TABLE IF NOT EXISTS networks
(
    network_id   UInt16,
    protocol     String,
    network_name String
)
ENGINE = ReplicatedMergeTree()
    PRIMARY KEY (network_id);


CREATE TABLE IF NOT EXISTS blobs
(
    network_id   UInt16,
    timestamp    DateTime,
    hash         String,
    key          String,
    blob_number  UInt64,
    rows         UInt32,
    columns      UInt32,
    sample_until DateTime
)
ENGINE = ReplicatedMergeTree()
    PRIMARY KEY (hash, blob_number)
TTL toDateTime(timestamp) + INTERVAL 180 DAY;


CREATE TABLE IF NOT EXISTS segments
(
    timestamp    DateTime,
    blob_number  UInt64,
    key          String,
    row          UInt32,
    column       UInt32,
    sample_until DateTime
)
ENGINE = ReplicatedMergeTree()
    PRIMARY KEY (blob_number, key)
TTL toDateTime(timestamp) + INTERVAL 180 DAY;


CREATE TABLE IF NOT EXISTS visits
(
    visit_round    UInt64,
    timestamp      DateTime,
    key            String,
    blob_number    UInt64,
    row            UInt32,
    column         UInt32,
    duration_ms    Int64,
    is_retrievable Bool,
    providers      UInt32,
    bytes          UInt32,
    error          String
)
ENGINE = ReplicatedMergeTree()
    PRIMARY KEY (visit_round)
TTL toDateTime(timestamp) + INTERVAL 180 DAY;