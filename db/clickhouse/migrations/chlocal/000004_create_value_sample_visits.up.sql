-- DO NOT EDIT: This file was generated with: just generate-local-clickhouse-migrations

-- Stores the results of the value sampling visist (FIND_VALUE)
CREATE TABLE value_visits (
    -- String representation of the visit type
    visit_type String,
    -- Number that identifies the visit round of the sampling
    visit_round UInt64,
    -- Name of the network (Protocol + Network) that the item belongs to
    network String,
    -- Timestamp of when the item was proposed (idealy), or when did akai become aware of it
    timestamp DateTime,
    -- Unique identifying key for the item (CID or DAS_CELL or DHT key)
    key String,
    -- Relational link to any chain or network block
    block_number UInt64,
    -- (Optional) row coordinates within the DAS schema
    das_row Nullable (UInt32),
    -- (Optional) column coordinates within the DAS schema
    das_column Nullable (UInt32),
    -- Number of milliseconds that akai spent doing the sampling
    duration_ms Int64,
    -- Boolean representation of the sampling result (key is retrievable or not)
    is_retrievable Bool,
    -- Number of bytes that we got in the sampling result
    bytes Int32,
    -- String representation of the error
    error String
) ENGINE = MergeTree () PRIMARY KEY (network, key, visit_round)
PARTITION BY
    toStartOfMonth (timestamp) TTL toDateTime (timestamp) + INTERVAL 180 DAY;
