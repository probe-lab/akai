-- DO NOT EDIT: This file was generated with: just generate-local-clickhouse-migrations

-- Stores all the information about any received block-info at the data sampler
CREATE TABLE blocks (
    -- Name of the network (Protocol + Network) that the block belongs to
    network String,
    -- Timestamp of when the block was proposed (idealy), or when did akai become aware of it
    timestamp DateTime,
    -- Number of the block in the given chain or network structure
    number UInt64,
    -- Unique Hash of the block in the chain or the network structure
    hash String,
    -- Extra identifying key for the block (CID or similars)
    key Nullable (String),
    -- Number of DAS rows that the block contains
    das_rows Nullable (UInt32),
    -- Number of DAS columns that the block contains
    das_columns Nullable (UInt32),
    -- Timestamp of when akai should stop caring about the block
    sample_until DateTime
) ENGINE = MergeTree () PRIMARY KEY (network, hash, number)
