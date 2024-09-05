package eventhandlers

import (
	"encoding/json"
	"fmt"
	"log"

	"github.com/streadway/amqp"
)

type MatchStatusChangedEvent struct {
	MatchId   string `json:"matchID"`
	NewStatus string `json:"status"`
	MatchName string `json:"matchName"`
	Round     string `json:"round"`
}

func PublishEvent(event MatchStatusChangedEvent) error {

	conn, err := amqp.Dial("amqp://guest:guest@10.133.66.153:5672/")

	if err != nil {
		return fmt.Errorf("failed to connect to RabbitMQ: %v", err)
	}
	defer conn.Close()
	// Open a channel
	ch, err := conn.Channel()
	if err != nil {
		return fmt.Errorf("failed to open a channel: %v", err)
	}
	defer ch.Close()

	// Declare an exchange
	err = ch.ExchangeDeclare(
		"match_status_exchange", // name
		"fanout",                // type
		true,                    // durable
		false,                   // auto-deleted
		false,                   // internal
		false,                   // no-wait
		nil,                     // arguments
	)
	if err != nil {
		return fmt.Errorf("failed to declare exchange: %v", err)
	}

	// Convert the event to JSON
	eventJSON, err := json.Marshal(event)
	if err != nil {
		return fmt.Errorf("failed to marshal event: %v", err)
	}

	// Publish the message to the exchange
	err = ch.Publish(
		"match_status_exchange", // exchange
		"",                      // routing key
		false,                   // mandatory
		false,                   // immediate
		amqp.Publishing{
			ContentType: "application/json",
			Body:        eventJSON,
		},
	)
	if err != nil {
		return fmt.Errorf("failed to publish message: %v", err)
	}

	log.Printf("Published event: %+v", event)

	return nil
}
