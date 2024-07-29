CREATE TABLE IF NOT EXISTS networks (
    network_id UInt16,
    protocol String,
    network_name String
)
ENGINE = MergeTree()
PRIMARY KEY (network_id);
