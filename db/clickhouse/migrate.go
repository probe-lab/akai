package clickhouse

import (
	"embed"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"

	_ "github.com/ClickHouse/clickhouse-go/v2"
	"github.com/golang-migrate/migrate/v4"
	_ "github.com/golang-migrate/migrate/v4/database/clickhouse"
	_ "github.com/golang-migrate/migrate/v4/source/file"

	log "github.com/sirupsen/logrus"
)

//go:embed migrations
var migrations embed.FS

func (s *ClickHouseDB) makeMigrations() error {
	log.Infof("applying database migrations...")

	tmpDir, err := os.MkdirTemp("", "akai")
	if err != nil {
		return fmt.Errorf("create tmp directory for migrations: %w", err)
	}
	defer func() {
		err = os.RemoveAll(tmpDir)
		if err != nil {
			log.WithFields(log.Fields{
				"tmpDir": tmpDir,
				"error":  err,
			}).Warn("Could not clean up tmp directory")
		}
	}()
	log.WithField("dir", tmpDir).Debugln("Created temporary directory")

	// point to the rigth migrations folder
	migrationsPath := ""
	conDetails := ""
	switch s.instanceType {
	case ClickhouseLocalInstance:
		migrationsPath = "migrations/chlocal"
		conDetails = s.conDetails.LocalMigrationDSN()
	case ClickhouseReplicatedInstance:
		migrationsPath = "migrations/chcluster"
		conDetails = s.conDetails.ReplicatedMigrationDSN()
	default:
		return fmt.Errorf("clickhouse doesn't support %s migrations", s.instanceType)
	}

	// copy migrations to tempDir
	err = fs.WalkDir(migrations, ".", func(path string, d fs.DirEntry, err error) error {
		join := filepath.Join(tmpDir, path)
		if d.IsDir() {
			return os.MkdirAll(join, 0o755)
		}

		data, readErr := migrations.ReadFile(path)
		if readErr != nil {
			return fmt.Errorf("read file: %w", readErr)
		}

		return os.WriteFile(join, data, 0o644)
	})
	if err != nil {
		return fmt.Errorf("create migration files: %w", err)
	}

	m, err := migrate.New("file://"+filepath.Join(tmpDir, migrationsPath), conDetails)
	if err != nil {
		return fmt.Errorf("applying migrations: %w", err)
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
