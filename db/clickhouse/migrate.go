package clickhouse

import (
	_ "github.com/ClickHouse/clickhouse-go/v2"

	"github.com/golang-migrate/migrate/v4"
	_ "github.com/golang-migrate/migrate/v4/database/clickhouse"
	_ "github.com/golang-migrate/migrate/v4/source/file"

	log "github.com/sirupsen/logrus"
)

func (s *ClickHouseDB) makeMigrations() error {
	log.Infof("applying database migrations...")
	// point to the migrations folder
	migrationDSN := s.conDetails.MigrationDSN()
	m, err := migrate.New("file://./db/clickhouse/migrations", migrationDSN)
	if err != nil {
		log.Warnf("Could not apply migrations from file://./db/clickhouse/migrations: %s", err)
		// HOT_FIX tests need the path to be relative
		m, err = migrate.New("file://migrations", migrationDSN)
		if err != nil {
			log.Errorf(err.Error())
			return err
		}
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
