package pubsub

import (
	"context"
	"encoding/json"
	"fmt"

	amqp "github.com/rabbitmq/amqp091-go"
)

type AckType int
type SimpleQueueType int

const (
	TransientQueue SimpleQueueType = iota
	DurableQueue                   = 1
)

const (
	Ack AckType = iota
	NackDiscard
	NackRequeue
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
	if simpleQueueType == int(TransientQueue) {
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
	handler func(T) AckType,
) error {
	chnl, que, err := DeclareAndBind(conn, exchange, queueName, key, simpleQueueType)
	if err != nil {
		return fmt.Errorf("could not declare and bind queue: %v", err)
	}

	msgs, err := chnl.Consume(que.Name, "", false, false, false, false, nil)
	if err != nil {
		return fmt.Errorf("could not consume messages: %v", err)
	}

	unmarshaller := func(data []byte) (T, error) {
		var target T
		err := json.Unmarshal(data, &target)
		return target, err
	}

	go func() {
		defer chnl.Close()
		for msg := range msgs {
			target, err := unmarshaller(msg.Body)
			if err != nil {
				fmt.Printf("could not unmarshal message: %v\n", err)
				continue
			}

			switch handler(target) {
			case Ack:
				msg.Ack(false)
				fmt.Println("Ack")
			case NackDiscard:
				msg.Nack(false, false)
				fmt.Println("NackDiscard")
			case NackRequeue:
				msg.Nack(false, true)
				fmt.Println("NackRequeue")
			}
		}
	}()

	return nil
}

func wrapDeclareBindError(err error) (*amqp.Channel, amqp.Queue, error) {
	return &amqp.Channel{}, amqp.Queue{}, err
}
