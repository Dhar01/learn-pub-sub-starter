package main

import (
	"log"

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

	username, err := gamelogic.ClientWelcome()
	if err != nil {
		log.Fatalf("can't get the username: %v", err)
	}

	chnl, _, err := pubsub.DeclareAndBind(
		conn,
		routing.ExchangePerilDirect,
		routing.PauseKey+"."+username,
		routing.PauseKey,
		pubsub.TransientQueue,
	)

	if err != nil {
		log.Fatalf("can't get channel and queue. Error: %v", err)
	}

	state := gamelogic.NewGameState(username)

	if err = pubsub.SubscribeJSON(
		conn,
		routing.ExchangePerilDirect,
		routing.PauseKey+"."+username,
		routing.PauseKey,
		pubsub.TransientQueue,
		handlerPause(state),
	); err != nil {
		log.Fatalf("Subscribe error: %v", err)
	}

	if err = pubsub.SubscribeJSON(
		conn,
		routing.ExchangePerilTopic,
		routing.ArmyMovesPrefix+"."+username,
		routing.ArmyMovesPrefix+".*",
		pubsub.TransientQueue,
		handlerMove(state, chnl),
	); err != nil {
		log.Fatalf("could not subscribe to army move: %v", err)
	}

	if err = pubsub.SubscribeJSON(
		conn,
		string(routing.ExchangePerilTopic),
		string(routing.WarRecognitionsPrefix),
		string(routing.WarRecognitionsPrefix)+".*",
		pubsub.DurableQueue,
		handlerWar(state),
	); err != nil {
		log.Fatalf("could not subscribe to war declaration: %v", err)
	}

	for {
		inp := gamelogic.GetInput()

		if len(inp) <= 0 {
			log.Println("Enter specific commands")
			continue
		}

		if inp[0] == "spawn" {
			if len(inp) != 3 {
				log.Println("invalid command format")
				continue
			}

			if err := state.CommandSpawn(inp); err != nil {
				log.Println("Error:", err)
				continue
			}
		} else if inp[0] == "move" {
			if len(inp) != 3 {
				log.Println("invalid command format")
				continue
			}

			moveData, err := state.CommandMove(inp)
			if err != nil {
				log.Println("Error:", err)
				continue
			}

			if err = pubsub.PublishJSON(
				chnl,
				routing.ExchangePerilTopic,
				routing.ArmyMovesPrefix+"."+username,
				moveData,
			); err != nil {
				log.Fatalf("Error: %v", err)
				continue
			}

			log.Println("move published successfully")
		} else if inp[0] == "status" {
			state.CommandStatus()
		} else if inp[0] == "help" {
			gamelogic.PrintClientHelp()
		} else if inp[0] == "spam" {
			log.Println("Spamming not allowed yet!")
		} else if inp[0] == "quit" {
			gamelogic.PrintQuit()
		} else {
			log.Println("invalid command input")
		}
	}

	// // shutting down
	// defer fmt.Println("system is shutting down...")
	// signalChan := make(chan os.Signal, 1)
	// signal.Notify(signalChan, os.Interrupt)
	// <-signalChan

}
