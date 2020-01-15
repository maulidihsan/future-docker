package main

import (
	"os/exec"
	"github.com/streadway/amqp"
	"fmt"
)

func main() {
    ReceiveMessage()
}

func ReceiveMessage() {
	conn, err := amqp.Dial("amqp://guest:guest@localhost:5672")
	if err != nil {
        panic("could not establish connection with RabbitMQ:" + err.Error())
	}
	defer conn.Close()

	channel, err := conn.Channel()
    if err != nil {
        panic("could not open RabbitMQ channel:" + err.Error())
    }
	defer channel.Close()

	_, err = channel.QueueDeclare(
		"test", // queue name
		true, // durable
		false, // delete when unused
		false, // exclusive
		false, // no-wait
		nil, // args
	)

    if err != nil {
        panic("error declaring the queue: " + err.Error())
	}

	err = channel.QueueBind(
		"test", // queue name
		"service", // routing key
		"events", // exchange name
		false,
		nil,
	)

    if err != nil {
        panic("error binding to the queue: " + err.Error())
	}
	// We consume data from the queue named Test using the channel we created in go.
	msgs, err := channel.Consume(
		"test", // queue name
		"", // consumer
		false, // auto-ack
		false, // exclusive
		false, // no-local
		false, // no-wait
		nil,
	)

    if err != nil {
        panic("error consuming the queue: " + err.Error())
	}

	forever := make(chan bool)

	go func() {
		for m := range msgs {
			msg := string(m.Body)
			command := exec.Command("docker", "service", "update", msg)
			err = command.Run()
			if err != nil {
				fmt.Sprintf("cmd.Run() failed with %s\n", err)
			}
			fmt.Println("restarted")
			m.Ack(false)
		}
	}()
	fmt.Printf(" [*] Waiting for messages. To exit press CTRL+C")
	<-forever
}