package config

import "fmt"

type DatabaseDetails struct {
	Driver   string
	Address  string
	User     string
	Password string
	Database string
	Params   string
}

func (d DatabaseDetails) String() string {
	return fmt.Sprintf(
		"%s://%s:%s@%s/%s%s",
		d.Driver,
		d.User,
		d.Password,
		d.Address,
		d.Database,
		d.Params,
	)
}
