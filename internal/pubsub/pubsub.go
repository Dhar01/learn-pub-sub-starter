package pubsub

import (
	"context"
	"encoding/json"

	amqp "github.com/rabbitmq/amqp091-go"
)

func PublishJSON[T any](ch *amqp.Channel, exchange, key string, val T) error {
	valByte, err := json.Marshal(val)
	if err != nil {
		return err
	}

	ctx := context.Background()
	mandatory := false
	immediate := false
	msg := amqp.Publishing{
		ContentType: "application/json",
		Body:        valByte,
	}

	if err := ch.PublishWithContext(ctx, exchange, key, mandatory, immediate, msg); err != nil {
		return err
	}

	return nil
}
