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
	ArgDeadLetterExchange = "x-dead-letter-exchange"
)

const (
	TransientQueue SimpleQueueType = iota
	DurableQueue
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

	return ch.PublishWithContext(context.Background(), exchange, key, false, false, amqp.Publishing{
		ContentType: "application/json",
		Body:        valByte,
	})
}

func DeclareAndBind(
	conn *amqp.Connection,
	exchange,
	queueName,
	key string,
	simpleQueueType SimpleQueueType,
) (*amqp.Channel, amqp.Queue, error) {
	chnl, err := conn.Channel()
	if err != nil {
		return wrapDeclareBindError(fmt.Errorf("couldn't create channel: %v", err))
	}

	queue, err := chnl.QueueDeclare(
		queueName,
		simpleQueueType == DurableQueue,
		simpleQueueType != DurableQueue,
		simpleQueueType != DurableQueue,
		false,
		amqp.Table{
			ArgDeadLetterExchange: "peril_dlx",
		},
	)
	if err != nil {
		return wrapDeclareBindError(fmt.Errorf("couldn't declare queue: %v", err))
	}

	if err = chnl.QueueBind(queueName, key, exchange, false, nil); err != nil {
		return wrapDeclareBindError(fmt.Errorf("couldn't bind queue: %v", err))
	}

	return chnl, queue, nil
}

func SubscribeJSON[T any](
	conn *amqp.Connection,
	exchange,
	queueName,
	key string,
	simpleQueueType SimpleQueueType,
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
	return nil, amqp.Queue{}, err
}
