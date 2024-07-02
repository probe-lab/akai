package config

import "fmt"

var (
	ClientName = "akai"
	Maintainer = "probelab"
)

func ComposeAkaiUserAgent() string {
	return fmt.Sprintf("%s/%s", Maintainer, ClientName)
}
