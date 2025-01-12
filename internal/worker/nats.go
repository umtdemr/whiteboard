package worker

import (
	"context"
	"fmt"
	"github.com/nats-io/nats.go"
	"github.com/nats-io/nats.go/jetstream"
	"time"
)

type NatsConfig struct {
	Url     string
	Stream  string
	Subject string
}

func SetupNATS(cfg NatsConfig) (*nats.Conn, jetstream.JetStream, jetstream.Stream, error) {
	nc, err := nats.Connect(cfg.Url, nats.Timeout(10*time.Second))
	if err != nil {
		return nil, nil, nil, fmt.Errorf("failed to connect to NATS: %w", err)
	}

	js, err := jetstream.New(nc)
	if err != nil {
		nc.Close()
		return nil, nil, nil, fmt.Errorf("failed to create JetStream context: %w", err)
	}

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	stream, err := js.CreateStream(ctx, jetstream.StreamConfig{
		Name:     "JOBS",
		Subjects: []string{cfg.Subject},
	})

	if err != nil {
		nc.Close()
		return nil, nil, nil, fmt.Errorf("failed to create stream :%w", err)
	}

	return nc, js, stream, nil
}
