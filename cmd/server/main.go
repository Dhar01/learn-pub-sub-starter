package main

import (
	"log"

	"github.com/bootdotdev/learn-pub-sub-starter/internal/pubsub"
	"github.com/bootdotdev/learn-pub-sub-starter/internal/routing"
	amqp "github.com/rabbitmq/amqp091-go"
)

func main() {
	const rabbitConnString = "amqp://guest:guest@localhost:5672/"

	conn, err := amqp.Dial(rabbitConnString)
	if err != nil {
		log.Fatalf("amqp connection error: %v", err)
	}

	defer conn.Close()

	chnl, err := conn.Channel()
	if err != nil {
		log.Fatalf("can't create a new channel: %v", err)
	}

	if err = pubsub.PublishJSON(chnl, routing.ExchangePerilDirect, routing.PauseKey, routing.PlayingState{
		IsPaused: true,
	}); err != nil {
		log.Fatalf("can't publish JSON: %v", err)
	}

	log.Println("Peril game server connected to RabbitMQ!")

	// // shutting down
	// defer fmt.Println("system is shutting down...")
	// signalChan := make(chan os.Signal, 1)
	// signal.Notify(signalChan, os.Interrupt)
	// <-signalChan

}
