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
		"/dns/bootnode.1.lightclient.mainnet.avail.so/tcp/37000/p2p/12D3KooW9x9qnoXhkHAjdNFu92kMvBRSiFBMAoC5NnifgzXjsuiM",
	}

	// Celestia bootnodes
	BootstrapNodesCelestiaMainnet = []string{
		"/dnsaddr/da-bootstrapper-1.celestia-bootstrap.net/p2p/12D3KooWSqZaLcn5Guypo2mrHr297YPJnV8KMEMXNjs3qAS8msw8",
		"/dnsaddr/da-bootstrapper-2.celestia-bootstrap.net/p2p/12D3KooWQpuTFELgsUypqp9N4a1rKBccmrmQVY8Em9yhqppTJcXf",
		"/dnsaddr/da-bootstrapper-3.celestia-bootstrap.net/p2p/12D3KooWKZCMcwGCYbL18iuw3YVpAZoyb1VBGbx9Kapsjw3soZgr",
		"/dnsaddr/da-bootstrapper-4.celestia-bootstrap.net/p2p/12D3KooWE3fmRtHgfk9DCuQFfY3H3JYEnTU3xZozv1Xmo8KWrWbK",
		"/dnsaddr/boot.celestia.pops.one/p2p/12D3KooWBBzzGy5hAHUQVh2vBvL25CKwJ7wbmooPcz4amQhzJHJq",
		"/dnsaddr/celestia.qubelabs.io/p2p/12D3KooWAzucnC7yawvLbmVxv53ihhjbHFSVZCsPuuSzTg6A7wgx",
		"/dnsaddr/celestia-bootstrapper.binary.builders/p2p/12D3KooWDKvTzMnfh9j7g4RpvU6BXwH3AydTrzr1HTW6TMBQ61HF",
	}
	BootstrapNodesCelestiaMocha4 = []string{
		"/dnsaddr/da-bootstrapper-1-mocha-4.celestia-mocha.com/p2p/12D3KooWCBAbQbJSpCpCGKzqz3rAN4ixYbc63K68zJg9aisuAajg",
		"/dnsaddr/da-bootstrapper-2-mocha-4.celestia-mocha.com/p2p/12D3KooWCUHPLqQXZzpTx1x3TAsdn3vYmTNDhzg66yG8hqoxGGN8",
		"/dnsaddr/mocha-boot.pops.one/p2p/12D3KooWDzNyDSvTBdKQAmnsUdAyQCQWwM3ReXTmPaaf6LzfNwRs",
		"/dnsaddr/celestia-mocha.qubelabs.io/p2p/12D3KooWQVmHy7JpfxpKZfLjvn12GjvMgKrWdsHkFbV2kKqQFBCG",
		"/dnsaddr/celestia-mocha4-bootstrapper.binary.builders/p2p/12D3KooWK6AYaPSe2EP99NP5G2DKwWLfMi6zHMYdD65KRJwdJSVU",
		"/dnsaddr/celestia-testnet-boot.01node.com/p2p/12D3KooWR923Tc8SCzweyaGZ5VU2ahyS9VWrQ8mDz56RbHjHFdzW",
		"/dnsaddr/celestia-mocha-boot.zkv.xyz/p2p/12D3KooWFdkhm7Ac6nqNkdNiW2g16KmLyyQrqXMQeijdkwrHqQ9J",
	}
)
