package containers_test

import (
	"database/sql"
	"os"
	"testing"

	"github.com/alextanhongpin/core/test/containers"
)

const postgresVersion = "15.1-alpine"

func TestMain(m *testing.M) {
	// Start the container.
	stop := containers.StartPostgres(postgresVersion, migrate)
	code := m.Run() // Run tests.
	stop()          // You can't defer this because os.Exit doesn't care for defer.
	os.Exit(code)
}

func migrate(db *sql.DB) error {
	_, err := db.Exec(`create table numbers(n int)`)
	if err != nil {
		return err
	}
	_, err = db.Exec(`create table names(name text)`)
	return err
}
