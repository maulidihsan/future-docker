package main

import (
	"flag"
	"log"
	"net/http"
	"encoding/json"
	"github.com/streadway/amqp"
    "github.com/gorilla/mux"
)

var AMQP *amqp.Connection

func get(w http.ResponseWriter, r *http.Request) {
    w.Header().Set("Content-Type", "application/json")
    w.WriteHeader(http.StatusOK)
    w.Write([]byte(`{"message": "get called"}`))
}

func post(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	var web Website
	err := json.NewDecoder(r.Body).Decode(&web)
    if err != nil {
        http.Error(w, err.Error(), http.StatusBadRequest)
        return
	}
	SendMessage(web)
    w.WriteHeader(http.StatusCreated)
    w.Write([]byte(`{"message": "request being processed"}`))
}

func put(w http.ResponseWriter, r *http.Request) {
    w.Header().Set("Content-Type", "application/json")
    w.WriteHeader(http.StatusAccepted)
    w.Write([]byte(`{"message": "put called"}`))
}

func delete(w http.ResponseWriter, r *http.Request) {
    w.Header().Set("Content-Type", "application/json")
    w.WriteHeader(http.StatusOK)
    w.Write([]byte(`{"message": "delete called"}`))
}

func notFound(w http.ResponseWriter, r *http.Request) {
    w.Header().Set("Content-Type", "application/json")
    w.WriteHeader(http.StatusNotFound)
    w.Write([]byte(`{"message": "not found"}`))
}

func SendMessage(newWeb Website) {
	channel, err := AMQP.Channel()
    if err != nil {
        panic("could not open RabbitMQ channel:" + err.Error())
    }
	defer channel.Close()

	err = channel.ExchangeDeclare(
		"events", // name
		"topic",  // type
		true,     // durable
		false,    // auto-deleted
		false,    // internal
		false,    // no-wait
		nil,      // arguments
	)
    if err != nil {
        panic(err)
	}

	msg, err := json.Marshal(newWeb)
	if err != nil {
		panic(err)
	}

	err = channel.Publish(
		"events", 	  // exchange
		"create.dir", // routing key
		false, 		  // mandatory
		false,  	  // immediate
		amqp.Publishing{
			Body: msg,
		},
	)
    if err != nil {
        panic("error publishing a message to the queue:" + err.Error())
    }
}

func main() {
	portPtr := flag.String("port", "3000", "listening port")
	rabbitmqPtr := flag.String("rabbitmq", "guest:guest@localhost:5672", "rabbit mq connection string (guest:guest@localhost:5672)")
	flag.Parse()
	conn, err := amqp.Dial("amqp://"+*rabbitmqPtr)
	if err != nil {
        panic("could not establish connection with RabbitMQ:" + err.Error())
	}
	AMQP = conn
	defer conn.Close()

	r := mux.NewRouter()
    r.HandleFunc("/", get).Methods(http.MethodGet)
    r.HandleFunc("/", post).Methods(http.MethodPost)
    r.HandleFunc("/", put).Methods(http.MethodPut)
    r.HandleFunc("/", delete).Methods(http.MethodDelete)
	r.HandleFunc("/", notFound)
	log.Print("serving on:"+ *portPtr)
	log.Fatal(http.ListenAndServe(":"+*portPtr, r))
}

type rabbitConn struct {
    rb *amqp.Connection
}

type Website struct {
    Username string `json:"username"`
	Email  string `json:"email"`
	SiteName string `json:"sitename"`
	SubDomain string `json:"subdomain"`
}