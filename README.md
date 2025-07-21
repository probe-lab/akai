![Akai ProbeLab Logo](./docs/banner-akai.svg)

# Akai
Akai is a generic Libp2p DHT item sampler originaly developed to perform Data Availability Sampling of DA Cells in the Avail DA DHT. However, it has been extended to support more DHT items in other p2p networks.

At the moment, these are the set of networks that it does support:
- Avail (Mainnet, Turing, Hex)
- Celestia (Mainnet, Mocha-4)
- IPFS (Amino DHT)

## Installation
The repository include a unified entry point through the `justfile`. The installation of the tool requires:
- [`Go >= 1.22`](https://go.dev/doc/install)
- ['Just'](https://github.com/casey/just)
- Connection to a ['Clickhouse DB'](https://clickhouse.com/)

Optional:
- [`Docker` / `Docker-compose`]() as the repo includes a local `docker-compose.yaml` to spaw a local clickhouse instance

To install the tools:
```
# compile the code and generate a binary under ./build folder
~$ just build
```
or
```
# install the tool on the user's $GOPATH
~$ just install
```

## How to run it
Akai has two separate ways of interacting with each DHT: the `find` command and the `daemon`.

### How to Find DHT items using Akai
The `find` command of Akai allows you to perform any of the supported DHT operations on each supported network (although not all support each operation):
- *Find Providers* for CIDs in the DHT
- *Find Peers* for Keys in the DHT
- *Find Values* for Keys in the DHT
- *Find Closest* peers for a given keys in the DHT
- Find the latest records and information of a Peer in the DHT

Instructions:
```
NAME:
   akai find - Performs a single operation for any Key from the given network's DHT

USAGE:
   akai find [command [command options]]

COMMANDS:
   closest    Finds the k closest peers for any Key at the given network's DHT
   providers  Finds the existing providers for any Key at the given network's DHT (For celestia namespaces, use FIND_PEERS)
   value      Finds the value for any Key at the given network's DHT
   peers      Finds the existing peers advertised for the given Key at the network's DHT
   peer-info  Finds all the existing information about a PeerID at the given network's DHT

OPTIONS:
   --network value, -n value  The network where the Akai will be launched. (default: [IPFS_AMINO][AVAIL_TURING, AVAIL_MAINNET, AVAIL_HEX][CELESTIA_MAINNET, CELESTIA_MOCHA-4][LOCAL_CUSTOM]) [$AKAI_PING_NETWORK]
   --key value, -k value      Key for the DHT operation that will be taken place (CID, peerID, key, etc.) [$AKAI_PING_KEY]
   --timeout value, -t value  Duration for the DHT find-providers operation (20s, 1min) (default: 20sec) [$AKAI_PING_TIMEOUT]
   --retries value, -r value  Number of attempts that akai will try to fetch the content (default: 3) [$AKAI_PING_RETRIES]
   --help, -h                 show help
```

Examples:
```
~$ ./build/akai find -n "IPFS_AMINO" -k "bafybeicn7i3soqdgr7dwnrwytgq4zxy7a5jpkizrvhm5mv6bgjd32wm3q4" -r 3 providers
INFO[0000] running ookla command...                      akai-log-format=text akai-log-level=info akai-metrics-address=127.0.0.1 akai-metrics-port=9080
INFO[0000] Starting metrics server                       address="http://127.0.0.1:9080/metrics"
INFO[0000] requesting key from given DHT...              key=bafybeicn7i3soqdgr7dwnrwytgq4zxy7a5jpkizrvhm5mv6bgjd32wm3q4 network=IPFS_AMINO operation=FIND_PROVIDERS retries=3 timeout=1m0s
...
INFO[0007] find providers done: (retry: 1)               duration_ms=1.362065263s error=no-error key=bafybeicn7i3soqdgr7dwnrwytgq4zxy7a5jpkizrvhm5mv6bgjd32wm3q4 num_providers=22 operation=FIND_PROVIDERS timestamp="2025-04-17 15:23:19.835624702 +0200 CEST m=+6.027451087"
INFO[0007] the providers...                              maddresses="[]" provider_0=12D3KooWRpTmvtYGgsMaWfH4E76KKoi8L8eNFcyNZ2E2TMcfx6wm
INFO[0007] the providers...                              maddresses="[]" provider_1=12D3KooWGKGfeL7GZtd1w8pxfCwwdMtVtjHynYKLYwJ9nJTwvXdN
INFO[0007] the providers...                              maddresses="[*****]" provider_2=12D3KooWHREL6tSzZWvXCZpNkkZnv4osd4M9ZonyBM7hom2L7tN3
...
```

### How to sample DHT items periodically
Akai also supports a second command that will check the availability of the items (or the operation we select) on a given frequency. This feature also reports back the results into a `clickhouse` database.

Instructions:
```
NAME:
   akai daemon - Runs the core of Akai's Data Sampler as a daemon

USAGE:
   akai daemon [command [command options]]

COMMANDS:
   avail     Tracks DAS cells from the Avail network and checks their availability in the DHT
   celestia  Tracks Celestia's Namespaces and checks the number of nodes supporting them at the DHT
   ipfs      Tracks IPFS CIDs over the API and checks their availability in the DHT

OPTIONS:
   --network value, -n value  The network where the Akai will be launched. (default: [IPFS_AMINO][AVAIL_TURING, AVAIL_MAINNET, AVAIL_HEX][CELESTIA_MAINNET, CELESTIA_MOCHA-4][LOCAL_CUSTOM]) [$AKAI_NETWORK]
   --db-driver value          Driver of the Database that will keep all the raw data (clickhouse-local, clickhouse-replicated) (default: "clickhouse") [$AKAI_DAEMON_DB_DRIVER]
   --db-address value         Address of the Database that will keep all the raw data (default: "127.0.0.1:9000") [$AKAI_DAEMON_DB_ADDRESS]
   --db-user value            User of the Database that will keep all the raw data (default: "username") [$AKAI_DAEMON_DB_USER]
   --db-password value        Password for the user of the given Database (default: "password") [$AKAI_DAEMON_DB_PASSWORD]
   --db-database value        Name of the Database that will keep all the raw data (default: "akai_test") [$AKAI_DAEMON_DB_DATABASE]
   --db-tls                   use or not use of TLS while connecting clickhouse (default: false) [$AKAI_DAEMON_DB_TLS]
   --samplers value           Number of workers the daemon will spawn to perform the sampling (default: 1000) [$AKAI_DAEMON_SAMPLERS]
   --sampling-timeout value   Timeout for each sampling operation at the daemon after we deduce it failed (default: 10s) [$AKAI_DAEMON_SAMPLING_TIMEOUT]
   --help, -h                 show help
```

## Maintainers

[@cortze](https://github.com/cortze)
[@kasteph](https://github.com/kasteph)
[@probe-lab](https://github.com/probe-lab)

## Contributing

Feel free to give us some feedback or to collaborate! [Open an issue](https://github.com/probe-lab/akai/issues/new) or submit PRs.
