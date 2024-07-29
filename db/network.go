package db

import (
	"fmt"
	"strings"
)

// DB table names

var NetworkTableName = "networks"
var networkBatcherSize = 1

type Network struct {
	NetworkID   uint16 `ch:"network_id"`
	Protocol    string `ch:"protocol"`
	NetworkName string `ch:"network_name"`
}

func (n Network) String() string {
	return fmt.Sprintf("%s_%s", n.Protocol, n.NetworkName)
}

func (n Network) FromString(s string) Network {
	parts := strings.Split(s, "_")
	protocol := strings.ToUpper(parts[0]) // the protocol goes always first
	network := "MAIN_NET"
	if len(parts) >= 2 {
		network = strings.ToUpper(parts[1])
	}
	return Network{
		Protocol:    protocol,
		NetworkName: network,
	}
}

func (n Network) IsComplete() bool {
	return n.NetworkID > 0 && n.Protocol != "" && n.NetworkName != ""
}

func (n Network) TableName() string {
	return NetworkTableName
}

func (n Network) QueryValues() map[string]any {
	return map[string]any{
		"network_id":   n.NetworkID,
		"protocol":     n.Protocol,
		"network_name": n.NetworkName,
	}
}

func (n Network) BatchingSize() int {
	return networkBatcherSize
}
