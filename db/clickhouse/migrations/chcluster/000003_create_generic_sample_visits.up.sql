-- Stores the results of the Generic visist (FIND_PROVIDERS / FIND_PEERS)
CREATE TABLE generic_visits (
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
    -- Number of milliseconds that akai spent doing the sampling
    duration_ms Int64,
    -- Number of items that we got in the sampling result
    response_items Int32,
    -- List of PeerIDs that we got at the sampling result
    peer_ids Array(String),
    -- String representation of the error
    error String
) ENGINE = ReplicatedMergeTree () PRIMARY KEY (network, key, visit_round)
PARTITION BY
    toStartOfMonth (timestamp) TTL toDateTime (timestamp) + INTERVAL 180 DAY;
