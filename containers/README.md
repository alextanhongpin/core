# containers

For integration testing with database, we use docker containers to spin up the infrastructure such as Postgres.



Just start the containers in the `TestMain`, `containers.Start<container-name>`.


For database such as postgres, an single-transaction driver has been implemented so that any writes to the database will be rollbacked at the end of the tests. This speeds up integration testing, as there is no need to truncate or drop and recreate the database.


```go
func TestMain(m *testing.M) {
	// Start the container.
	stop := containers.StartPostgres(postgresVersion)

	code := m.Run()

	// You can't defer this because os.Exit doesn't care for defer.
	stop()

	os.Exit(code)
}

// Run you tests.
func TestPostgres(t *testing.T) {
	db := containers.PostgresDB(t)

	var n int
	err := db.QueryRow("select 1 + 1").Scan(&n)
	if err != nil {
		t.Error(err)
	}

	want := 2
	got := n
	if want != got {
		t.Fatalf("sum: want %d, got %d", want, got)
	}
}
```


If you want to run migrations, seeds or fixtures, you can do it in `TestMain` too:


```go
func migrate(db *sql.DB) error {
	// ...
}

func TestMain(m *testing.M) {
	// Start the container.
	stop := containers.StartPostgres(postgresVersion, migrate)

	code := m.Run()

	// You can't defer this because os.Exit doesn't care for defer.
	stop()

	os.Exit(code)
}
```


Now, for every test, it will rollback to state before any inserts/updates are done to the database.
