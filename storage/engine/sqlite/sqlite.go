package sqlite

import (
	_ "modernc.org/sqlite/vec" // Registers the sqlite-vec extension automatically

	"cmp"
	"database/sql"
	"database/sql/driver"
	"encoding/json"
	"errors"
	"fmt"
	"reflect"
	"time"

	"modernc.org/sqlite"
)

func init() {
	sqlite.MustRegisterFunction("json_containment", &sqlite.FunctionImpl{
		NArgs:        2,
		VolatileArgs: true,
		Scalar: func(ctx *sqlite.FunctionContext, args []driver.Value) (driver.Value, error) {
			if len(args) != 2 {
				return false, errors.New("invalid args")
			}
			if args[0] == nil || args[1] == nil {
				return false, nil
			}

			var superset, subset map[string]any
			err := cmp.Or(
				json.Unmarshal([]byte(args[0].(string)), &superset),
				json.Unmarshal([]byte(args[1].(string)), &subset),
			)
			if err != nil {
				return nil, fmt.Errorf("unmarshaling json_containment: %w", err)
			}
			return jsonContainment(superset, subset), nil
		},
	})
}

func New(path string) (*sql.DB, error) {
	// Replace with "sqlite" if using a pure Go driver, or "sqlite3" for CGO
	dsn := path + "?" +
		"_pragma=journal_mode(WAL)" + // Enable Write-Ahead Logging for concurent reading.
		"&_pragma=busy_timeout(5000)" + // Wait up to 5s if locked instead of failing instantly.
		"&_pragma=synchronous(NORMAL)" + // Balance safety and high write througput.
		"&_pragma=foreign_keys(ON)" + // Force relational constraint integrity.
		"&_pragma=temp_store(MEMORY)" + // Store temporary tables/indices in RAM
		"&_pragma=cache_size(-64000)" + // Set page cache size to -64MB
		"&_time_integer_format=unix_nano" +
		"&_inttotime=1" +
		"&_texttotime=1" +
		"&_txlock=immediate"
	db, err := sql.Open("sqlite", dsn)
	if err != nil {
		return nil, err
	}

	db.SetMaxOpenConns(1)                  // Strict limit to prevent write deadlocks.
	db.SetMaxIdleConns(1)                  // Match open connections to avoid recyling overhead.
	db.SetConnMaxLifetime(time.Hour)       // Clean up stale connections gracefully.
	db.SetConnMaxIdleTime(2 * time.Minute) // Free resources promptly.

	if err := db.Ping(); err != nil {
		return nil, errors.Join(err, db.Close())
	}

	return db, nil
}

func jsonContainment(superset, subset map[string]any) bool {
	for k, subsetVal := range subset {
		supersetVal, ok := superset[k]
		if !ok {
			return false
		}
		if !reflect.DeepEqual(subsetVal, supersetVal) {
			return false
		}
	}

	return true
}
