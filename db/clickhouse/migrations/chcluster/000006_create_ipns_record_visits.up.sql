-- Stores the results of IPNS record visits
CREATE TABLE ipns_record_visits (
    -- Number that identifies the visit round of the sampling
    visit_round UInt64,
    -- Timestamp of when the IPNS record was visited
    timestamp DateTime,
    -- Unique identifier of the IPNS record
    record String,
    -- Type of IPNS record (e.g., "ipns-record", "dns-link")
    record_type String,
    -- Quorum target for the DHT lookup
    quorum UInt8,
    -- Sequence number of the found record
    seq_number UInt32,
    -- Seconds that the item will be available in the network
    ttl_s Float64,
    -- Whether the signature was verified or verifiable
    is_valid Bool,
    -- Whether the item was retrieved from the network or not
    is_retrievable Bool,
    -- Resulting CID/Link representation
    result String,
    -- Number of nanoseconds that the operation took
    duration_ms Int64,
    -- String representation of the error
    error String
) ENGINE = ReplicatedMergeTree() PRIMARY KEY (visit_round, record)
PARTITION BY
    toStartOfMonth (timestamp) TTL toDateTime (timestamp) + INTERVAL 180 DAY;
