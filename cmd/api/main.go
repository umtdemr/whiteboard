package main

import (
	"context"
	"expvar"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/julienschmidt/httprouter"
	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"
	"github.com/umtdemr/wb-backend/internal/config"
	"github.com/umtdemr/wb-backend/internal/data"
	db "github.com/umtdemr/wb-backend/internal/db/sqlc"
	"github.com/umtdemr/wb-backend/internal/worker"
	"github.com/umtdemr/wb-backend/internal/ws"
	"os"
	"runtime"
	"sync"
	"time"
)

const version = "1.0.0"

type application struct {
	config       config.Config
	models       data.Models
	wg           sync.WaitGroup
	jobPublisher worker.Publisher
	router       *httprouter.Router
	wsHub        *ws.Hub
}

func main() {
	zerolog.TimeFieldFormat = zerolog.TimeFormatUnix

	conf, err := config.LoadConfig(".")

	if err != nil {
		log.Fatal().Msgf("failed to create config %s", err.Error())
	}

	if conf.Environment == "dev" {
		log.Logger = log.Output(zerolog.ConsoleWriter{Out: os.Stderr})
	}

	conn, err := pgxpool.New(context.Background(), conf.DBSource)

	if err != nil {
		log.Fatal().Msgf("could not establish db connection %s", err)
	}

	dbStore := db.NewStore(conn)
	defer conn.Close()
	dbData := dbStore.(*db.SQLStore)

	if err != nil {
		log.Fatal().Msgf("could not establish db connection %s", err)
	}

	expvar.NewString("version").Set(version)

	expvar.Publish("goroutines", expvar.Func(func() any {
		return runtime.NumGoroutine()
	}))

	expvar.Publish("database", expvar.Func(func() any {
		type database struct {
			MaxConn   int32 `json:"max_conn"`
			TotalConn int32 `json:"total_conn"`
			IdleConns int32 `json:"idle_conns"`
		}
		stats := dbData.DB.Stat()
		return database{
			MaxConn:   stats.MaxConns(),
			TotalConn: stats.TotalConns(),
			IdleConns: stats.IdleConns(),
		}
	}))

	expvar.Publish("timestamp", expvar.Func(func() any {
		return time.Now().Unix()
	}))

	nc, err := worker.SetupNats(conf.NatsServerUrl)

	if err != nil {
		log.Fatal().Msgf("failed to setup NATS: %v", err)
	}
	defer nc.Close()

	js, stream, err := worker.SetupJetStream(nc)
	if err != nil {
		log.Fatal().Msgf("failed to setup JetStream: %v", err)
	}

	jobPublisher := worker.NewWorker(js, stream)
	models := data.NewModels(dbStore)

	app := &application{
		config:       conf,
		models:       models,
		jobPublisher: jobPublisher,
		router:       httprouter.New(),
		wsHub:        ws.NewHub(models, nc),
	}

	go app.wsHub.Run()

	err = app.serve()
	if err != nil {
		log.Error().Msg(err.Error())
		os.Exit(1)
	}
}
