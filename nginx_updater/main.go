package main

import (
	"flag"
	"os/exec"
	"github.com/isayme/go-amqp-reconnect/rabbitmq"
	"fmt"
)

func main() {
	rabbitmqPtr := flag.String("rabbitmq", "guest:guest@localhost:5672", "rabbit mq connection string (guest:guest@localhost:5672)")
	flag.Parse()
	ReceiveMessage(*rabbitmqPtr)
}

func ReceiveMessage(connString string) {
	conn, err := rabbitmq.Dial("amqp://"+connString)
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
		"restart", // queue name
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
		"restart", // queue name
		"service_events", // routing key
		"events", // exchange name
		false,
		nil,
	)

    if err != nil {
        panic("error binding to the queue: " + err.Error())
	}
	// We consume data from the queue named Test using the channel we created in go.
	msgs, err := channel.Consume(
		"restart", // queue name
		"service_restart", // consumer
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
			fmt.Println("Restarting "+ msg)
			command := exec.Command("docker", "service", "update", "--force", msg)
			err = command.Run()
			if err != nil {
				fmt.Println("cmd.Run() failed with "+ err.Error())
			}
			fmt.Println("restarted")
			m.Ack(false)
		}
	}()
	fmt.Println(" [*] Waiting for messages. To exit press CTRL+C")
	<-forever
}