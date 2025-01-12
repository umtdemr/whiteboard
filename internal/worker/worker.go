package worker

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"github.com/nats-io/nats.go"
	"github.com/nats-io/nats.go/jetstream"
	"github.com/rs/zerolog/log"
	"time"
)

type JobType string

const (
	JobTypeEmail JobType = "email"
)

type Job struct {
	Type JobType `json:"type"`
	Data any     `json:"data"`
}

type Publisher interface {
	EnqueueJob(ctx context.Context, job Job) error
}

type Processor interface {
	ProcessJobs(ctx context.Context, handler func(Job) error) error
}

type Worker struct {
	js     jetstream.JetStream
	stream jetstream.Stream
}

// Ensure Worker implements both interfaces
var _ Publisher = (*Worker)(nil)
var _ Processor = (*Worker)(nil)

func NewWorker(js jetstream.JetStream, stream jetstream.Stream) *Worker {
	return &Worker{js, stream}
}

func (w *Worker) EnqueueJob(ctx context.Context, job Job) error {
	jobData, err := json.Marshal(job)
	if err != nil {
		return fmt.Errorf("failed to marshal job: %w", err)
	}

	_, err = w.js.Publish(ctx, "jobs."+string(job.Type), jobData)
	if err != nil {
		return fmt.Errorf("failed to publish job: %w", err)
	}

	return nil
}

func (w *Worker) ProcessJobs(ctx context.Context, handler func(job Job) error) error {
	cons, err := w.stream.CreateOrUpdateConsumer(ctx, jetstream.ConsumerConfig{
		Name:          "job-processor",
		Durable:       "job-processor",
		AckPolicy:     jetstream.AckExplicitPolicy,
		DeliverPolicy: jetstream.DeliverAllPolicy,
		FilterSubject: "jobs.>",
		MaxDeliver:    5,
		MaxAckPending: 1,
		AckWait:       60 * time.Second,
	})

	if err != nil {
		log.Error().Err(err).Msg("failed to create or update consumer")
	}

	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		default:
			messages, err := cons.Fetch(1, jetstream.FetchMaxWait(5*time.Second))
			if err != nil {
				if errors.Is(err, nats.ErrTimeout) {
					continue
				}
				log.Info().Msgf("error fetching message: %v", err)
				continue
			}

			for msg := range messages.Messages() {
				var job Job
				err := json.Unmarshal(msg.Data(), &job)
				if err != nil {
					log.Printf("error unmarshaling job: %w", err)
					msg.Nak()
					continue
				}

				err = handler(job)
				if err != nil {
					log.Printf("error processing job: %v\n", err)
					msg.Nak()
				} else {
					msg.Ack()
				}
			}
		}
	}
}
