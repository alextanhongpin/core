package postgres

import (
	"database/sql"
	"fmt"

	"github.com/uptrace/bun"
	"github.com/uptrace/bun/dialect/pgdialect"
	"github.com/uptrace/bun/driver/pgdriver"
	"github.com/uptrace/bun/extra/bundebug"
)

func NewBun() *bun.DB {
	user := mustEnv("DB_USER")
	pass := mustEnv("DB_PASS")
	name := mustEnv("DB_NAME")
	port := mustEnv("DB_PORT")
	host := mustEnv("DB_HOST")
	app := mustEnv("APP_NAME")

	sqldb := sql.OpenDB(pgdriver.NewConnector(
		pgdriver.WithUser(user),
		pgdriver.WithAddr(fmt.Sprintf("%s:%s", host, port)),
		pgdriver.WithPassword(pass),
		pgdriver.WithDatabase(name),
		pgdriver.WithApplicationName(app),
		pgdriver.WithInsecure(true),
	))

	db := bun.NewDB(sqldb, pgdialect.New())

	db.AddQueryHook(bundebug.NewQueryHook(
		bundebug.WithVerbose(true),
		bundebug.FromEnv("BUNDEBUG"),
	))

	return db
}
