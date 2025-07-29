-- Stores the results of peer info visits
CREATE TABLE peer_info_visits (
    -- Number that identifies the visit round of the sampling
    visit_round UInt64,
    -- Timestamp of when the peer was visited
    timestamp DateTime,
    -- Name of the network (Protocol + Network) that the item belongs to
    network String,
    -- Unique identifier of the peer
    peer_id String,
    -- Number of milliseconds that akai spent doing the sampling
    duration_ms Int64,
    -- Agent version reported by the peer
    agent_version String,
    -- List of protocols supported by the peer
    protocols Array(String),
    -- Version of the main protocol
    protocol_version String,
    -- Network addresses where peer can be reached
    multi_addresses Array(String),
    -- String representation of the error
    error String
) ENGINE = ReplicatedMergeTree() PRIMARY KEY (network, peer_id, visit_round)
PARTITION BY
    toStartOfMonth (timestamp) TTL toDateTime (timestamp) + INTERVAL 180 DAY;
