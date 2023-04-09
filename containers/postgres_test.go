package containers_test

import (
	"os"
	"testing"

	"github.com/alextanhongpin/go-core-microservice/containers"
)

func TestMain(m *testing.M) {
	// Start the container.
	stop := containers.Postgres("15.1-alpine")
	code := m.Run()

	// You can't defer this because os.Exit doesn't care for defer.
	stop()

	os.Exit(code)
}

func TestPostgres(t *testing.T) {
	db, err := containers.DB()
	if err != nil {
		t.Error(err)
	}
	t.Cleanup(func() {
		db.Close()
	})

	var n int
	err = db.QueryRow("select 1 + 1").Scan(&n)
	if err != nil {
		t.Error(err)
	}
	t.Log("got n:", n)
}
