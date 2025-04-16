-- DO NOT EDIT: This file was generated with: just generate-local-clickhouse-migrations
-- Stores all the information about any sampling item that arrived at the data sampler
CREATE TABLE items (
    -- Timestamp of when the item was proposed (idealy), or when did akai become aware of it
    timestamp DateTime,
    -- Name of the network (Protocol + Network) that the item belongs to
    network String,
    -- String representation of the item type
    item_type String,
    -- String representation of the type of sampling method that it requires
    sample_type String,
    -- Relational link to any chain or network block
    block_link UInt64,
    -- Unique identifying key for the item (CID or DAS_CELL or DHT key)
    key String,
    -- Unique Hash of the item in the chain or the network structure
    hash String,
    -- (Optional) row coordinates within the DAS schema
    das_row Nullable (UInt32),
    -- (Optional) column coordinates within the DAS schema
    das_column Nullable (UInt32),
    -- (Optional) any string representation or extra info that could be metadata
    metadata Nullable (String),
    -- Bolean flag that determines whether we are just storing the information or we do want to trace it down
    traceable Bool,
    -- Timestamp of when akai should stop caring about the item
    sample_until DateTime
) ENGINE = MergeTree () PRIMARY KEY (network, key) TTL toDateTime (timestamp) + INTERVAL 180 DAY;
