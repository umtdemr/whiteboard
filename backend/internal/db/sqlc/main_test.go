package db

import (
	"context"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/rs/zerolog/log"
	"github.com/umtdemr/wb-backend/internal/config"
	"os"
	"testing"
)

var testStore Store

func TestMain(m *testing.M) {
	conf, err := config.LoadConfig("../../../")
	if err != nil {
		log.Fatal().Msgf("cannot load config: %s", err)
	}

	poolConfig, err := pgxpool.ParseConfig(conf.DBSource)
	poolConfig.MaxConns = 1
	if err != nil {
		log.Fatal().Msg("failed to parse config")
	}

	connPool, err := pgxpool.NewWithConfig(context.Background(), poolConfig)
	if err != nil {
		log.Fatal().Msgf("cannot connect to db: %s", err)
	}

	testStore = NewStore(connPool)
	os.Exit(m.Run())
}
