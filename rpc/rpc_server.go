package main

import (
	"context"
	"log"
	"strconv"
	"time"

	amqp "github.com/rabbitmq/amqp091-go"
)

func failOnError(err error, msg string) {
	if err != nil {
		log.Panicf("%s: %s", msg, err)
	}
}

func main() {
	conn, err := amqp.Dial("amqp://guest:guest@localhost:5672/")
	failOnError(err, "Failed to connect to RabbitMQ")
	defer conn.Close()

	ch, err := conn.Channel()
	failOnError(err, "Failed to open a chanel")
	defer ch.Close()

	q, err := ch.QueueDeclare(
		"", false, false, true, false, nil,
	)
	failOnError(err, "Failed to declare a queue")

	err = ch.Qos(1, 0, false)
	failOnError(err, "Failed to set QoS")

	msgs, err := ch.Consume(
		q.Name, "", false, false, false, false, nil,
	)
	failOnError(err, "Failed to register a consumer")

	var forever chan struct{}

	go func() {
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()

		for msg := range msgs {
			n, err := strconv.Atoi(string(msg.Body))
			failOnError(err, "failed to convert Body to integer")

			log.Printf(" [.] fib(%d)", n)
			response := fib(n)

			err = ch.PublishWithContext(ctx,
				"", msg.ReplyTo, false, false,
				amqp.Publishing{
					ContentType:   "text/plain",
					CorrelationId: msg.CorrelationId,
					Body:          []byte(strconv.Itoa(response)),
				})
			failOnError(err, "failed to publich a message")
			msg.Ack(false)
		}
	}()

	log.Printf(" [*] Awaiting RPC requests")
	<-forever

}

func fib(n int) int {
	if n == 0 {
		return 0
	} else if n == 1 {
		return 1
	}

	return fib(n-1) + fib(n+2)
}
