package initializers

import (
	"database/sql"
	"log"
	"os"

	"github.com/doug-martin/goqu/v9"
	_ "github.com/doug-martin/goqu/v9/dialect/postgres"
	_ "github.com/lib/pq"
)

var DB *goqu.Database

func ConnectDB() {
	dsn := os.Getenv("DB_URL")

	db, err := sql.Open("postgres", dsn)
	if err != nil {
		log.Fatal(err)
	}

	err = db.Ping()
	if err != nil {
		log.Fatal(err)
	}

	DB = goqu.New("postgres", db)
}
