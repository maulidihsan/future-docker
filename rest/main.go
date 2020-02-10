package main

import (
	"flag"
	"log"
	"os"
	"fmt"
	"time"
	"math/rand"
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
	resp, err := json.Marshal(GetAllUser())
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

	exists, err := IsUserExists(web.Username)
	if err != nil {
        http.Error(w, err.Error(), http.StatusInternalServerError)
        return
    }
	if (exists) {
		w.WriteHeader(http.StatusOK)
		w.Header().Set("Content-Type", "application/json")
		w.Write([]byte(`{"message": "username already exist"}`))
		return
	}
    web.Password = GenPasswd(8)
    err = CreateDB(web)
    if err != nil {
        http.Error(w, err.Error(), http.StatusInternalServerError)
        return
    }

    err = AddUser(web)
    if err != nil {
        http.Error(w, err.Error(), http.StatusInternalServerError)
        return
    }
	web.Action = "install"
	SendMessage(web)
	w.WriteHeader(http.StatusCreated)
	w.Header().Set("Content-Type", "application/json")
	w.Write([]byte(`{"message": "request being processed"}`))
	return
}

func postUpdate(w http.ResponseWriter, r *http.Request) {
	var web Website
	err := json.NewDecoder(r.Body).Decode(&web)
    if err != nil {
        http.Error(w, err.Error(), http.StatusBadRequest)
        return
    }

	exists, err := IsSubDomainExists(web.SubDomain)
	if err != nil {
        http.Error(w, err.Error(), http.StatusInternalServerError)
        return
    }
    if (exists) {
		w.WriteHeader(http.StatusOK)
		w.Header().Set("Content-Type", "application/json")
		w.Write([]byte(`{"message": "subdomain already exist"}`))
		return
    }
    web.Email, err = GetEmail(web)
    if err != nil {
        http.Error(w, err.Error(), http.StatusInternalServerError)
        return
    }
    subdomain, err := UpdateDomain(web)
    if err != nil {
        http.Error(w, err.Error(), http.StatusInternalServerError)
        return
    }
	web.CurrentSubDomain = subdomain
	web.Action = "update_domain"
	SendMessage(web)
	w.WriteHeader(http.StatusCreated)
	w.Header().Set("Content-Type", "application/json")
    w.Write([]byte(`{"message": "request being processed"}`))
}

func postResetPassword(w http.ResponseWriter, r *http.Request) {
	var web Website
	err := json.NewDecoder(r.Body).Decode(&web)
    if err != nil {
        http.Error(w, err.Error(), http.StatusBadRequest)
        return
    }
    newPass, err := ResetPassword(web)
    if err != nil {
        http.Error(w, err.Error(), http.StatusInternalServerError)
        return
    }
	web.Password = newPass
	web.Action = "reset_password"
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

    err = DeleteUser(web)
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
		w.Header().Set("Access-Control-Allow-Origin", "http://website.ku")
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

const charset = "abcdefghijklmnopqrstuvwxyz" + "ABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789"
func GenPasswd(length int) string {
	var seededRand *rand.Rand = rand.New(rand.NewSource(time.Now().UnixNano()))
	b := make([]byte, length)
	for i := range b {
		b[i] = charset[seededRand.Intn(len(charset))]
	}
	return string(b)
}

func GetEmail(web Website) (string, error) {
	var email string
	row := DBCon.QueryRow(fmt.Sprintf("SELECT email FROM vsftpd.users WHERE username='%s';", web.Username))
	err := row.Scan(&email)
	if err != nil {
		return "", err
	}
	return email, nil
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

func IsUserExists(username string) (bool, error) {
	var exists bool
	err := DBCon.QueryRow(fmt.Sprintf("SELECT exists (SELECT * FROM vsftpd.users WHERE username='%s')", username)).Scan(&exists)
	if err != nil {
		return false, err
	}
	return exists, nil
}

func IsSubDomainExists(domain string) (bool, error) {
	var exists bool
	err := DBCon.QueryRow(fmt.Sprintf("SELECT exists (SELECT * FROM vsftpd.users WHERE subdomain='%s')", domain)).Scan(&exists)
	if err != nil {
		return false, err
	}
	return exists, nil
}

func CreateDB(web Website) error {
	_, err := DBCon.Exec(fmt.Sprintf("CREATE DATABASE IF NOT EXISTS wp_%s", web.Username))
	if err != nil {
		return err
	}
	_, err = DBCon.Exec(fmt.Sprintf("GRANT ALL ON wp_%s.* to %s@'%%' identified by '%s'", web.Username, web.Username, web.Password))
	if err != nil {
		return err
	}
	return nil
}

func AddUser(web Website) error {
	_, err := DBCon.Exec(fmt.Sprintf("INSERT INTO vsftpd.users (username, password, email, subdomain) VALUES ('%s',md5('%s'), '%s', '%s')", web.Username, web.Password, web.Email, web.Username))
	if err != nil {
		return err
	}
	return nil
}

func UpdateDomain(web Website) (string, error) {
	var subdomain string
	row := DBCon.QueryRow(fmt.Sprintf("SELECT subdomain FROM vsftpd.users WHERE username='%s';", web.Username))
	err := row.Scan(&subdomain)
	if err != nil {
		return "",err
	}
	_, err = DBCon.Exec(fmt.Sprintf("UPDATE vsftpd.users SET subdomain='%s' WHERE username='%s'", web.SubDomain, web.Username))
	if err != nil {
		return "",err
	}
	return subdomain, nil
}

func ResetPassword(web Website) (string, error) {
	newPass := GenPasswd(8)
	_, err := DBCon.Exec(fmt.Sprintf("UPDATE vsftpd.users SET password = md5('%s') WHERE username='%s';", newPass, web.Username))
	if err != nil {
		return "", err
	}

	_, err = DBCon.Exec(fmt.Sprintf("ALTER USER '%s'@'%%' IDENTIFIED BY '%s';", web.Username, newPass))
	if err != nil {
		return "", err
	}
	return newPass, nil
}

func DeleteUser(web Website) error {
	_, err := DBCon.Exec(fmt.Sprintf("DELETE FROM vsftpd.users WHERE username='%s'", web.Username))
	if err != nil {
		return err
	}

	_, err = DBCon.Exec(fmt.Sprintf("DROP DATABASE wp_%s", web.Username))
	if err != nil {
		return err
	}

	_, err = DBCon.Exec(fmt.Sprintf("DROP USER '%s'@'%%';", web.Username))
	if err != nil {
		return err
	}
	return nil
}

func GetAllUser() []Website {
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
	r.HandleFunc("/reset", corsHandler(postResetPassword)).Methods(http.MethodPost, http.MethodOptions)
    r.HandleFunc("/delete", corsHandler(postDelete)).Methods(http.MethodPost, http.MethodOptions)
	log.Print("serving on:"+ *portPtr)
	loggedHandler := handlers.LoggingHandler(os.Stdout, r)
	log.Fatal(http.ListenAndServe(":"+*portPtr, loggedHandler))
}

type Website struct {
    Username string `json:"username"`
	Email  string `json:"email"`
	SubDomain string `json:"subdomain"`
	CurrentSubDomain string `json: "current_subdomain"`
	Action string `json: "action"`
	Password string `json: "password"`
}