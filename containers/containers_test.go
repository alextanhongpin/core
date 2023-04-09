package containers_test

import (
	"os"
	"testing"

	"github.com/alextanhongpin/go-core-microservice/containers"
)

const postgresVersion = "15.1-alpine"

func TestMain(m *testing.M) {
	// Start the container.
	stopPostgres := containers.StartPostgres(postgresVersion)
	stopPostgresBun := containers.StartPostgresBun(postgresVersion)

	code := m.Run()

	// You can't defer this because os.Exit doesn't care for defer.
	stopPostgres()
	stopPostgresBun()

	os.Exit(code)
}
