package containers_test

import (
	"os"
	"testing"

	"github.com/alextanhongpin/core/test/containers"
)

const postgresVersion = "15.1-alpine"

func TestMain(m *testing.M) {
	// Start the container.
	stop := containers.StartPostgres(postgresVersion)
	code := m.Run() // Run tests.
	stop()          // You can't defer this because os.Exit doesn't care for defer.
	os.Exit(code)
}
