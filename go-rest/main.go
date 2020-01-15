package main

import (
	"log"
	"fmt"
	"net/http"
	"encoding/json"
	"github.com/streadway/amqp"
    "github.com/gorilla/mux"
)

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
	fmt.Println(web)
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
	fmt.Println("checked")
    if err != nil {
        panic("error publishing a message to the queue:" + err.Error())
    }
}

func main() {
    r := mux.NewRouter()
    r.HandleFunc("/", get).Methods(http.MethodGet)
    r.HandleFunc("/", post).Methods(http.MethodPost)
    r.HandleFunc("/", put).Methods(http.MethodPut)
    r.HandleFunc("/", delete).Methods(http.MethodDelete)
    r.HandleFunc("/", notFound)
    log.Fatal(http.ListenAndServe(":3000", r))
}

type Website struct {
    Username string `json:"username"`
	Email  string `json:"email"`
	SiteName string `json:"sitename"`
}