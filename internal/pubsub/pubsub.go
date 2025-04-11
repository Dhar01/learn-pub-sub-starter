package pubsub

import (
	"context"
	"encoding/json"
	"fmt"

	amqp "github.com/rabbitmq/amqp091-go"
)

const (
	Transient = 0
	Durable   = 1
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

func DeclareAndBind(
	conn *amqp.Connection,
	exchange,
	queueName,
	key string,
	simpleQueueType int,
) (*amqp.Channel, amqp.Queue, error) {
	chnl, err := conn.Channel()
	if err != nil {
		return wrapDeclareBindError(err)
	}

	durable := true
	autoDelete := false
	exclusive := false

	// if queueType is transient
	if simpleQueueType == Transient {
		durable = false
		autoDelete = true
		exclusive = true
	}

	queue, err := chnl.QueueDeclare(queueName, durable, autoDelete, exclusive, false, nil)
	if err != nil {
		return wrapDeclareBindError(err)
	}

	if err = chnl.QueueBind(queueName, key, exchange, false, nil); err != nil {
		return wrapDeclareBindError(err)
	}

	return chnl, queue, nil
}

func SubscribeJSON[T any](
	conn *amqp.Connection,
	exchange,
	queueName,
	key string,
	simpleQueueType int,
	handler func(T),
) error {
	chnl, que, err := DeclareAndBind(conn, exchange, queueName, key, simpleQueueType)
	if err != nil {
		return err
	}

	ch, err := chnl.Consume(que.Name, "", false, false, false, false, nil)
	if err != nil {
		return err
	}

	go func() {
		for delivery := range ch {
			var msg T
			if err := json.Unmarshal(delivery.Body, &msg); err != nil {
				fmt.Println("Error unmarshaling the message:", err)
				continue
			}

			handler(msg)
			delivery.Ack(false)
		}
	}()

	return nil
}

func wrapDeclareBindError(err error) (*amqp.Channel, amqp.Queue, error) {
	return &amqp.Channel{}, amqp.Queue{}, err
}
