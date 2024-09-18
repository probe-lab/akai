package config

var (
	// Turing testnet for the Avail Network
	// reference: https://raw.githubusercontent.com/availproject/avail/main/misc/genesis/testnet.turing.chain.spec.raw.json
	BootstrapNodesAvailTurin = []string{
		"/dns/bootnode-turing-001.avail.so/tcp/30333/p2p/12D3KooWRYCs192er73R2oyE4QWkKdtoRk1fL28tR3immzxTQQAq",
		"/dns/bootnode-turing-002.avail.so/tcp/30333/p2p/12D3KooWLxzYHmGXpRnN4rAKkkZVzFgpGuqKofPggUnWz4iUR5yT",
		"/dns/bootnode-turing-003.avail.so/tcp/30333/p2p/12D3KooWR8yPHmBZxHPoZrjjvCZEq5mJh9h7yewgM4zGgUVEANcG",
		"/dns/bootnode-turing-004.avail.so/tcp/30333/p2p/12D3KooWKkLgiqxweQ1QCbJrDKeg5HCktZMM1kd81F4jMMTnBtD6",
		"/dns/bootnode-turing-005.avail.so/tcp/30333/p2p/12D3KooWQRTgqX2j2HMideuPKUg12yeCbbp6cCi6xtf4cACYSiKb",
		"/dns/bootnode-turing-006.avail.so/tcp/30333/p2p/12D3KooWPNmovc7n6mET56bmpB8tM6XtjcRb8qBir3sTBWk5cttK",
		"/dns/bootnode-turing-007.avail.so/tcp/30333/p2p/12D3KooWFzqtue91MrdmqVRddA48dPYQVAFBtcFkf5HrmgiqUJ6F",
		"/dns/bootnode-turing-008.avail.so/tcp/30333/p2p/12D3KooWMHRgZhDMUbN55NQcXB19Ww6SnL8shz1un3K6X3K2XoAh",
		"/dns/bootnode-turing-009.avail.so/tcp/30333/p2p/12D3KooWQLdPYPEHRGAJ4J6dGYoiKyjm36VhaSCJkvKAuJBohJYi",
	}

	BootstrapNodesAvailHex = []string{
		"/dns/bootnode.1.lightclient.hex.avail.so/tcp/37000/p2p/12D3KooWBMwfo5qyoLQDRat86kFcGAiJ2yxKM63rXHMw2rDuNZMA",
	}

	BootstrapNodesAvailMainnet = []string{
		"/dns/bootnode-mainnet-001.avail.so/tcp/30333/p2p/12D3KooWBXk3rcfKkvd1YbJ8fHPH4WZy34QCw8Czrvqf6cbmrdKh",
		"/dns/bootnode-mainnet-002.avail.so/tcp/30333/p2p/12D3KooWMJt1z4ap2UacerW7vJprr7Mqws5sqmJXZqg8XpZswiwF",
		"/dns/bootnode-mainnet-003.avail.so/tcp/30333/p2p/12D3KooWK85zotBy9jjgVhHzsvntAWWzRxTosF3RxwnJ9zJsGQbi",
		"/dns/bootnode-mainnet-004.avail.so/tcp/30333/p2p/12D3KooWFyPvzGbTH7qbB3foBDDfwtn4LJsb4ryvTGGCokUyYDm7",
		"/dns/bootnode-mainnet-005.avail.so/tcp/30333/p2p/12D3KooWFB6wBdwS1aacu3QZdmZb2sPDiVxP9iLFecLBiHKZELCX",
		"/dns/bootnode-mainnet-006.avail.so/tcp/30333/p2p/12D3KooWLVJtN3hpUJYPjQzGKiHnbw6fpD5vvH5EkztdP1VAAgrj",
		"/dns/bootnode-mainnet-007.avail.so/tcp/30333/p2p/12D3KooWPfsS6gAYoEiB8EaWbHdrbd6ugcru2kGguz56B9PxD4HT",
		"/dns/bootnode-mainnet-008.avail.so/tcp/30333/p2p/12D3KooWSYiWfRCigRwWU9UzVJfZRz3SkCpYySH9w5ZydXUJWVuJ",
		"/dns/bootnode-mainnet-009.avail.so/tcp/30333/p2p/12D3KooWH1dPXxYyxwYjhbrjJ3PUJht9DyQkgdiJyiTtKVx5Voh4",
		"/dns/bootnode-mainnet-010.avail.so/tcp/30333/p2p/12D3KooWLeXFyp1Ghm7oCt1EghhT39wnsNhZWT1jYEWTHn2pHU3E",
	}

	// Celestia bootnodes
	BootstrapNodesCelestiaMainnet = []string{
		"/dns4/da-bridge-1.celestia-bootstrap.net/tcp/2121/p2p/12D3KooWSqZaLcn5Guypo2mrHr297YPJnV8KMEMXNjs3qAS8msw8",
		"/dns4/da-bridge-2.celestia-bootstrap.net/tcp/2121/p2p/12D3KooWQpuTFELgsUypqp9N4a1rKBccmrmQVY8Em9yhqppTJcXf",
		"/dns4/da-bridge-3.celestia-bootstrap.net/tcp/2121/p2p/12D3KooWSGa4huD6ts816navn7KFYiStBiy5LrBQH1HuEahk4TzQ",
		"/dns4/da-bridge-4.celestia-bootstrap.net/tcp/2121/p2p/12D3KooWHBXCmXaUNat6ooynXG837JXPsZpSTeSzZx6DpgNatMmR",
		"/dns4/da-bridge-5.celestia-bootstrap.net/tcp/2121/p2p/12D3KooWDGTBK1a2Ru1qmnnRwP6Dmc44Zpsxi3xbgFk7ATEPfmEU",
		"/dns4/da-bridge-6.celestia-bootstrap.net/tcp/2121/p2p/12D3KooWLTUFyf3QEGqYkHWQS2yCtuUcL78vnKBdXU5gABM1YDeH",
		"/dns4/da-full-1.celestia-bootstrap.net/tcp/2121/p2p/12D3KooWKZCMcwGCYbL18iuw3YVpAZoyb1VBGbx9Kapsjw3soZgr",
		"/dns4/da-full-2.celestia-bootstrap.net/tcp/2121/p2p/12D3KooWE3fmRtHgfk9DCuQFfY3H3JYEnTU3xZozv1Xmo8KWrWbK",
		"/dns4/da-full-3.celestia-bootstrap.net/tcp/2121/p2p/12D3KooWK6Ftsd4XsWCsQZgZPNhTrE5urwmkoo5P61tGvnKmNVyv",
	}

	BootstrapNodesCelestiaMocha4 = []string{
		"/dns4/da-bridge-mocha-4.celestia-mocha.com/tcp/2121/p2p/12D3KooWCBAbQbJSpCpCGKzqz3rAN4ixYbc63K68zJg9aisuAajg",
		"/dns4/da-bridge-mocha-4-2.celestia-mocha.com/tcp/2121/p2p/12D3KooWK6wJkScGQniymdWtBwBuU36n6BRXp9rCDDUD6P5gJr3G",
		"/dns4/da-full-1-mocha-4.celestia-mocha.com/tcp/2121/p2p/12D3KooWCUHPLqQXZzpTx1x3TAsdn3vYmTNDhzg66yG8hqoxGGN8",
		"/dns4/da-full-2-mocha-4.celestia-mocha.com/tcp/2121/p2p/12D3KooWR6SHsXPkkvhCRn6vp1RqSefgaT1X1nMNvrVjU2o3GoYy",
	}
)
