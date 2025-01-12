package main

import (
	"context"
	"fmt"
	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"
	"github.com/umtdemr/wb-backend/internal/config"
	"github.com/umtdemr/wb-backend/internal/mailer"
	"github.com/umtdemr/wb-backend/internal/worker"
	"os"
	"os/signal"
	"syscall"
	"time"
)

type backgroundWorker struct {
	mailer mailer.Mailer
}

func main() {
	conf, err := config.LoadConfig(".")

	if err != nil {
		log.Fatal().Msgf("failed to create config %s", err.Error())
	}

	if conf.Environment == "dev" {
		log.Logger = log.Output(zerolog.ConsoleWriter{Out: os.Stderr})
	}
	natsConfig := worker.NatsConfig{
		Url:     conf.NatsServerUrl,
		Stream:  "JOBS",
		Subject: "jobs.>",
	}

	nc, js, stream, err := worker.SetupNATS(natsConfig)
	if err != nil {
		log.Fatal().Msgf("failed to setup NATS: %v", err)
	}
	defer nc.Close()

	jobProcessor := worker.NewWorker(js, stream)

	ctx, cancel := context.WithCancel(context.Background())

	bgWorker := &backgroundWorker{
		mailer: mailer.New(conf.SmtpHost, conf.SmtpPort, conf.SmtpUsername, conf.SmtpPassword, "wb <no-reply@wb.net>"),
	}

	go func() {
		if err = jobProcessor.ProcessJobs(ctx, bgWorker.handleJob); err != nil {
			log.Fatal().Msgf("failed to start processing jobs: %v", err)
		}
	}()

	log.Info().Msg("worker is running and waiting for jobs")

	// graceful shutdown
	c := make(chan os.Signal, 1)
	signal.Notify(c, os.Interrupt, syscall.SIGTERM)
	<-c

	log.Info().Msg("worker shutting down")
	cancel()
	time.Sleep(2 * time.Second)
}

func (w *backgroundWorker) handleJob(job worker.Job) error {
	log.Info().Msgf("processing job: Type=%s, Data=%v", job.Type, job.Data)

	switch job.Type {
	case worker.JobTypeEmail:
		return w.sendEmail(job.Data)
	default:
		log.Info().Msgf("unknown job type: %s", job.Type)
	}

	return nil
}

// sendEmail handles email sending job
func (w *backgroundWorker) sendEmail(data interface{}) error {
	// since json serializes structs as map, reflect email job as map
	dataMap, ok := data.(map[string]interface{})
	if !ok {
		return fmt.Errorf("invalid email job data: %v", data)
	}
	to, _ := dataMap["to"].(string)
	tmplFile, _ := dataMap["tmpl_file"].(string)
	tmplData, _ := dataMap["tmpl_data"].(map[string]interface{})
	log.Info().Msg("send email")

	return w.mailer.Send(to, tmplFile, tmplData)
}
