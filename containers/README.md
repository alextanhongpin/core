# containers

```go
package main_test

import (
	"database/sql"
	"os"
	"testing"

	"github.com/alextanhongpin/go-core-microservice/containers"
)

var db *sql.DB

func TestMain(m *testing.M) {
	// Start the container.
	var stop func()
	db, stop = containers.Postgres("15.1-alpine")
	code := m.Run()

	// You can't defer this because os.Exit doesn't care for defer.
	stop()

	os.Exit(code)
}

func TestDocker(t *testing.T) {
	// Do sth ...
	var n int
	err := db.QueryRow("select 1 + 1").Scan(&n)
	if err != nil {
		t.Error(err)
	}
	t.Log("got n:", n)
}
```
