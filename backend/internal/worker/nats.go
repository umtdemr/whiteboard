package worker

import (
	"context"
	"fmt"
	"github.com/nats-io/nats.go"
	"github.com/nats-io/nats.go/jetstream"
	"time"
)

func SetupNats(url string) (*nats.Conn, error) {
	nc, err := nats.Connect(
		url,
		nats.Timeout(10*time.Second),
		nats.RetryOnFailedConnect(true),
		nats.MaxReconnects(-1),
		nats.ReconnectWait(3*time.Second),
	)

	return nc, err
}

func SetupJetStream(nc *nats.Conn) (jetstream.JetStream, jetstream.Stream, error) {
	js, err := jetstream.New(nc)
	if err != nil {
		nc.Close()
		return nil, nil, fmt.Errorf("failed to create JetStream context: %w", err)
	}

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	stream, err := js.CreateStream(ctx, jetstream.StreamConfig{
		Name:     "JOBS",
		Subjects: []string{"jobs.>"},
	})

	if err != nil {
		nc.Close()
		return nil, nil, fmt.Errorf("failed to create stream :%w", err)
	}

	return js, stream, nil
}
