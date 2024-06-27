package postgres

import (
	"context"
	"fmt"
	"github.com/jackc/pgx/v4"
	"github.com/viciious/mika/config"
	"github.com/viciious/mika/store"
	"github.com/viciious/mika/util"
	log "github.com/sirupsen/logrus"
	"io/ioutil"
	"os"
	"testing"
)

func TestTorrentDriver(t *testing.T) {
	db, err := pgx.Connect(context.Background(), makeDSN(config.Store))
	if err != nil {
		t.Skipf("failed to connect to postgres torrent store: %s", err.Error())
		return
	}
	setupDB(t, db)
	store.TestStore(t, &Driver{db: db, ctx: context.Background()})
}

func clearDB(db *pgx.Conn) {
	ctx := context.Background()
	for _, table := range []string{"peers", "torrent", "users", "whitelist"} {
		q := fmt.Sprintf(`drop table if exists %s cascade;`, table)
		if _, err := db.Exec(ctx, q); err != nil {
			log.Panicf("Failed to prep database: %s", err.Error())
		}
	}
}

func setupDB(t *testing.T, db *pgx.Conn) {
	clearDB(db)
	schema := util.FindFile("store/postgres/schema.sql")
	b, err := ioutil.ReadFile(schema)
	if err != nil {
		panic("Cannot read schema file")
	}
	if _, err := db.Exec(context.Background(), string(b)); err != nil {
		log.Panicf("Failed to setupDB: %s", err)
	}
	t.Cleanup(func() {
		clearDB(db)
	})
}

func TestMain(m *testing.M) {
	if err := config.Read("mika_testing_postgres"); err != nil {
		log.Info("Skipping database tests, failed to find config: mika_testing_postgres.yaml")
		os.Exit(0)
		return
	}
	if config.General.RunMode != "test" {
		log.Info("Skipping database tests, not running in testing mode")
		os.Exit(0)
		return
	}
	exitCode := m.Run()
	os.Exit(exitCode)
}
