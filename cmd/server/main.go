package main

import (
	"log"
	"os"

	"github.com/bootdotdev/learn-pub-sub-starter/internal/gamelogic"
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
	log.Println("Peril game server connected to RabbitMQ!")

	chnl, err := conn.Channel()
	if err != nil {
		log.Fatalf("can't create a new channel: %v", err)
	}

	_, _, err = pubsub.DeclareAndBind(conn, routing.ExchangePerilTopic, routing.GameLogSlug, routing.GameLogSlug+".*", pubsub.DurableQueue)
	if err != nil {
		log.Fatalf("can't get channel and queue, Error: %v", err)
	}

	var paused bool
	gamelogic.PrintServerHelp()

	for {
		inp := gamelogic.GetInput()
		if len(inp) == 0 {
			continue
		}

		if inp[0] == routing.PauseKey {
			log.Println("sending message: pause")
			paused = true
		} else if inp[0] == "resume" {
			log.Println("sending message: resume")
			paused = false
		} else if inp[0] == "quit" {
			log.Println("Exiting the peril server game")
			os.Exit(0)
		} else {
			log.Println("invalid command input")
		}

		if err = pubsub.PublishJSON(chnl, routing.ExchangePerilDirect, routing.PauseKey, routing.PlayingState{
			IsPaused: paused,
		}); err != nil {
			log.Fatalf("can't publish JSON: %v", err)
		}

	}

	// // shutting down
	// defer fmt.Println("system is shutting down...")
	// signalChan := make(chan os.Signal, 1)
	// signal.Notify(signalChan, os.Interrupt)
	// <-signalChan

}
