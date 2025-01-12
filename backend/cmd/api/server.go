package main

import (
	"context"
	"errors"
	"fmt"
	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"
	stdlog "log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"
)

type zerologBridge struct {
	logger zerolog.Logger
}

func (zb zerologBridge) Write(p []byte) (n int, err error) {
	zb.logger.Error().Msg(string(p))
	return len(p), nil
}

func (app *application) serve() error {
	serverLogger := log.With().Str("component", "http_server").Logger()
	bridge := &zerologBridge{logger: serverLogger}

	srv := &http.Server{
		Addr:         fmt.Sprintf(":%s", app.config.Port),
		Handler:      app.routes(),
		IdleTimeout:  time.Minute,
		ReadTimeout:  10 * time.Second,
		WriteTimeout: 30 * time.Second,
		ErrorLog:     stdlog.New(bridge, "", 0),
	}

	shutdownError := make(chan error)
	go func() {
		quit := make(chan os.Signal, 1)
		signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
		s := <-quit

		log.Info().Msgf("caught signal: %s", s.String())

		ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
		defer cancel()

		err := srv.Shutdown(ctx)

		if err != nil {
			shutdownError <- err
		}

		// Log a message to say that we're waiting for any background goroutines to
		// complete their tasks.
		log.Info().Msgf("completing background tasks, addr %v", srv.Addr)

		app.wg.Wait()

		shutdownError <- nil
		os.Exit(0)
	}()

	log.Info().Msgf("server started at %v", app.config.Port)

	err := srv.ListenAndServe()

	if !errors.Is(err, http.ErrServerClosed) {
		return err
	}

	err = <-shutdownError

	if err != nil {
		return err
	}

	log.Info().Msgf("stopped server - addr %v", srv.Addr)
	return nil
}
