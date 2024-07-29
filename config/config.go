package config

import (
	"fmt"
	"time"
)

// commands

// Root
type Root struct {
	Verbose    bool
	LogLevel   string
	LogFormat  string
	LogSource  bool
	LogNoColor bool
}

type Ping struct {
	Network string
	Key     string
	Timeout time.Duration
}

type Service struct {
	Network string
}

// Database defines the basic configuration of a database
type Database struct {
	// File path to the JSON output directory
	JSONOut string

	// Determines the host address of the database.
	DatabaseHost string

	// Determines the port of the database.
	DatabasePort int

	// Determines the name of the database that should be used.
	DatabaseName string

	// Determines the password with which we access the database.
	DatabasePassword string

	// Determines the username with which we access the database.
	DatabaseUser string

	// Postgres SSL mode (should be one supported in https://www.postgresql.org/docs/current/libpq-ssl.html)
	DatabaseSSLMode string

	// Set the maximum idle connections for the database handler.
	MaxIdleConns int
}

// DatabaseSourceName returns the data source name string to be put into the sql.Open method.
func (c *Database) DatabaseSourceName() string {
	return fmt.Sprintf(
		"host=%s port=%d dbname=%s user=%s password=%s sslmode=%s",
		c.DatabaseHost,
		c.DatabasePort,
		c.DatabaseName,
		c.DatabaseUser,
		c.DatabasePassword,
		c.DatabaseSSLMode,
	)
}
