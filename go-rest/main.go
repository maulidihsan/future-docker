package main

import (
	"flag"
	"log"
	"fmt"
	"net/http"
	"encoding/json"
	"database/sql"
	"github.com/streadway/amqp"
    "github.com/gorilla/mux"
)

var AMQP *amqp.Connection
var DBCon *sql.DB

func get(w http.ResponseWriter, r *http.Request) {
    w.Header().Set("Content-Type", "application/json")
	
	resp, err := json.Marshal(GetAllWeb())
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
	}
	w.WriteHeader(http.StatusOK)
    w.Write(resp)
}

func postCreate(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	var web Website
	err := json.NewDecoder(r.Body).Decode(&web)
    if err != nil {
        http.Error(w, err.Error(), http.StatusBadRequest)
        return
	}
	web.Action = "create"
	SendMessage(web)
    w.WriteHeader(http.StatusCreated)
    w.Write([]byte(`{"message": "request being processed"}`))
}

func postUpdate(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	var web Website
	err := json.NewDecoder(r.Body).Decode(&web)
    if err != nil {
        http.Error(w, err.Error(), http.StatusBadRequest)
        return
	}
	web.Action = "update"
	SendMessage(web)
    w.WriteHeader(http.StatusCreated)
    w.Write([]byte(`{"message": "request being processed"}`))
}

func postDelete(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	var web Website
	err := json.NewDecoder(r.Body).Decode(&web)
    if err != nil {
        http.Error(w, err.Error(), http.StatusBadRequest)
        return
	}
	web.Action = "delete"
	SendMessage(web)
    w.WriteHeader(http.StatusCreated)
    w.Write([]byte(`{"message": "request being processed"}`))
}

func SendMessage(newWeb Website) {
	channel, err := AMQP.Channel()
    if err != nil {
        panic("could not open RabbitMQ channel:" + err.Error())
    }
	defer channel.Close()

	err = channel.ExchangeDeclare(
		"events", // name
		"direct",  // type
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
		"web_events", // routing key
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

func GetAllWeb() []Website {
	rows, err := DBCon.Query("SELECT username, subdomain FROM vsftpd.users")
	if err != nil {
		fmt.Println(err.Error())
	}
	defer rows.Close()
	var results []Website
	for rows.Next() {
		web := Website{}
		err = rows.Scan(&web.Username, &web.SubDomain)
		if err != nil {
			fmt.Println(err.Error())
		}
		results = append(results, web)
	}
	return results
}

func main() {
	portPtr := flag.String("port", "3000", "listening port")
	rabbitmqPtr := flag.String("rabbitmq", "guest:guest@localhost:5672", "rabbit mq connection string (guest:guest@localhost:5672)")
	dbauthPtr := flag.String("mysql-auth", "root:password", "msyql username:password")
	dbhostPtr := flag.String("db", "127.0.0.1:3006", "mysql host (127.0.0.1:3306) root:password@tcp(127.0.0.1:3306)")
	
	flag.Parse()

	DBCon, err := sql.Open("mysql", fmt.Sprintf("%s@tcp(%s)/", *dbauthPtr, *dbhostPtr))
	if err != nil {
		fmt.Println(err.Error())
	}
	defer DBCon.Close()

	AMQP, err := amqp.Dial("amqp://"+*rabbitmqPtr)
	if err != nil {
        panic("could not establish connection with RabbitMQ:" + err.Error())
	}
	defer AMQP.Close()

	r := mux.NewRouter()
    r.HandleFunc("/", get).Methods(http.MethodGet)
    r.HandleFunc("/create", postCreate).Methods(http.MethodPost)
    r.HandleFunc("/update", postUpdate).Methods(http.MethodPost)
    r.HandleFunc("/delete", postDelete).Methods(http.MethodPost)
	log.Print("serving on:"+ *portPtr)
	log.Fatal(http.ListenAndServe(":"+*portPtr, r))
}

type Website struct {
    Username string `json:"username"`
	Email  string `json:"email"`
	SubDomain string `json:"subdomain"`
	Action string
}