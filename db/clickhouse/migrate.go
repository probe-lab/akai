package clickhouse

import (
	"fmt"

	_ "github.com/ClickHouse/clickhouse-go/v2"

	"github.com/golang-migrate/migrate/v4"
	_ "github.com/golang-migrate/migrate/v4/database/clickhouse"
	_ "github.com/golang-migrate/migrate/v4/source/file"

	log "github.com/sirupsen/logrus"
)

func (s *ClickHouseDB) makeMigrations() error {
	log.Infof("applying database migrations...")
	// point to the migrations folder
	s.conDetails.Params = s.conDetails.Params + "?x-multi-statement=true" // allow multistatements for the migrations
	m, err := migrate.New("file://./db/clickhouse/migrations", s.conDetails.String())
	if err != nil {
		fmt.Println("error - ")
		log.Errorf(err.Error())
		return err
	}
	// bring up the migrations to the last version
	if err := m.Up(); err != nil {
		if err != migrate.ErrNoChange {
			return err
		}
	}
	// close the migrator
	connErr, dbErr := m.Close()
	if connErr != nil {
		return connErr
	}
	if dbErr != nil {
		return dbErr
	}
	return err
}
