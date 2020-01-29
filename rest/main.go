package main

import (
	"flag"
	"log"
	"os"
	"fmt"
	"net/http"
	"encoding/json"
	"database/sql"
	"github.com/streadway/amqp"
	"github.com/gorilla/mux"
	"github.com/gorilla/handlers"
	_ "github.com/go-sql-driver/mysql"
)

var AMQP *amqp.Connection
var DBCon *sql.DB

func get(w http.ResponseWriter, r *http.Request) {
	resp, err := json.Marshal(GetAllWeb())
	if err != nil {
		fmt.Println(err.Error())
		w.WriteHeader(http.StatusInternalServerError)
	}

	w.WriteHeader(http.StatusOK)
	w.Header().Set("Content-Type", "application/json")
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

	w.WriteHeader(http.StatusCreated)
	w.Header().Set("Content-Type", "application/json")
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

	w.WriteHeader(http.StatusCreated)
	w.Header().Set("Content-Type", "application/json")
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

	w.WriteHeader(http.StatusCreated)
	w.Header().Set("Content-Type", "application/json")
    w.Write([]byte(`{"message": "request being processed"}`))
}

func corsHandler(h http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Access-Control-Allow-Origin", "*")
		w.Header().Set("Access-Control-Allow-Methods", "POST, GET, OPTIONS")
		w.Header().Set("Access-Control-Allow-Headers", "Content-Type")
		if (r.Method == "OPTIONS") {
			w.WriteHeader(http.StatusOK)
			return
		} else {
			h.ServeHTTP(w,r)
		}
	}
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
		"events",         // exchange
		"web_events", // routing key
		false,            // mandatory
		false,            // immediate
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
    r.HandleFunc("/", corsHandler(get)).Methods(http.MethodGet, http.MethodOptions)
    r.HandleFunc("/create", corsHandler(postCreate)).Methods(http.MethodPost, http.MethodOptions)
    r.HandleFunc("/update", corsHandler(postUpdate)).Methods(http.MethodPost, http.MethodOptions)
    r.HandleFunc("/delete", corsHandler(postDelete)).Methods(http.MethodPost, http.MethodOptions)
	log.Print("serving on:"+ *portPtr)
	loggedHandler := handlers.LoggingHandler(os.Stdout, r)
	log.Fatal(http.ListenAndServe(":"+*portPtr, loggedHandler))
}

type Website struct {
    Username string `json:"username"`
	Email  string `json:"email"`
	SubDomain string `json:"subdomain"`
	Action string `json: "action"`
}