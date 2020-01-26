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
	_ "github.com/go-sql-driver/mysql"
)

var AMQP *amqp.Connection
var DBCon *sql.DB


func addHeader(w *http.ResponseWriter) {
	(*w).Header().Set("Access-Control-Allow-Origin", "*")
	(*w).Header().Set("Content-Type", "application/json")
}

func get(w http.ResponseWriter, r *http.Request) {

	resp, err := json.Marshal(GetAllWeb())
	if err != nil {
		fmt.Println(err.Error())
		w.WriteHeader(http.StatusInternalServerError)
	}
	addHeader(&w)
	w.WriteHeader(http.StatusOK)
    w.Write(resp)
}

func postCreate(w http.ResponseWriter, r *http.Request) {
	var web Website
	err := json.NewDecoder(r.Body).Decode(&web)
    if err != nil {
        http.Error(w, err.Error(), http.StatusBadRequest)
        return
	}
	web.Action = "create"
	SendMessage(web)

	addHeader(&w)
    w.WriteHeader(http.StatusCreated)
    w.Write([]byte(`{"message": "request being processed"}`))
}

func postUpdate(w http.ResponseWriter, r *http.Request) {
	var web Website
	err := json.NewDecoder(r.Body).Decode(&web)
    if err != nil {
        http.Error(w, err.Error(), http.StatusBadRequest)
        return
	}
	web.Action = "update"
	SendMessage(web)

	addHeader(&w)
    w.WriteHeader(http.StatusCreated)
    w.Write([]byte(`{"message": "request being processed"}`))
}

func postDelete(w http.ResponseWriter, r *http.Request) {
	var web Website
	err := json.NewDecoder(r.Body).Decode(&web)
    if err != nil {
        http.Error(w, err.Error(), http.StatusBadRequest)
        return
	}
	web.Action = "delete"
	SendMessage(web)

	addHeader(&w)
    w.WriteHeader(http.StatusCreated)
    w.Write([]byte(`{"message": "request being processed"}`))
}

func SendMessage(newWeb Website) {
	channel, err := AMQP.Channel()
    if err != nil {
        panic("could not open RabbitMQ channel:" + err.Error())
    }
	defer channel.Close()
	msg, err := json.Marshal(newWeb)
	if err != nil {
		panic("cannot convert to json "+ err.Error())
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
	var tmp Website
	for rows.Next() {
		err = rows.Scan(&tmp.Username, &tmp.SubDomain)
		if err != nil {
			fmt.Println(err.Error())
		}
		results = append(results, tmp)
	}
	return results
}

func main() {
	portPtr := flag.String("port", "3000", "listening port")
	rabbitmqPtr := flag.String("rabbitmq", "guest:guest@localhost:5672", "rabbit mq connection string (guest:guest@localhost:5672)")
	dbauthPtr := flag.String("mysql-auth", "root:password", "msyql username:password")
	dbhostPtr := flag.String("mysql-host", "127.0.0.1:3006", "mysql host (127.0.0.1:3306)")
	
	flag.Parse()
	var err error
	DBCon, err = sql.Open("mysql", fmt.Sprintf("%s@tcp(%s)/", *dbauthPtr, *dbhostPtr))
	if err != nil {
		fmt.Println(err.Error())
	}
	defer DBCon.Close()

	AMQP, err = amqp.Dial("amqp://"+*rabbitmqPtr)
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
	Action string `json: "action"`
}