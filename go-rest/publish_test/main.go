package main

import (
	"fmt"
	"flag"
	"strings"
	"os"
	"os/exec"
	"math/rand"
	"time"
	"path/filepath"
	"database/sql"
	"encoding/json"
	_ "github.com/go-sql-driver/mysql"
	"github.com/streadway/amqp"
)


func main() {
	rabbitmqPtr := flag.String("rabbitmq", "guest:guest@localhost:5672", "rabbit mq connection string (guest:guest@localhost:5672)")
	stackNamePtr := flag.String("stack-name", "shared", "docker stack name")
	dbauthPtr := flag.String("mysql-auth", "root:password", "msyql username:password")
	dbhostPtr := flag.String("db", "127.0.0.1:3006", "mysql host (127.0.0.1:3306) root:password@tcp(127.0.0.1:3306)")
	dirPathPtr := flag.String("dir", "./", "wordpress root path")
	confPathPtr := flag.String("conf-dir", "./", "nginx conf path")
	flag.Parse()
    ReceiveMessage(*rabbitmqPtr, *stackNamePtr, *dbauthPtr, *dbhostPtr, *dirPathPtr, *confPathPtr)
}

func ReceiveMessage(r string, stack string, dbAuth string, dbHost string, dirPath string, confPath string) {
	conn, err := amqp.Dial("amqp://"+r)
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
		"create.dir", // routing key
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
			website := Website{}
			json.Unmarshal([]byte(msg), &website)
			website.Password = String(8)
			CreateDB(website, dbAuth, dbHost)
			AddUser(website, dbAuth, dbHost)
			SetupScript(website, dbHost, dirPath, confPath)
			SendMessage(conn, stack+"_nginx")
			m.Ack(false)
		}
	}()
	fmt.Printf(" [*] Waiting for messages. To exit press CTRL+C")
	<-forever
}

const charset = "abcdefghijklmnopqrstuvwxyz" + "ABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789"
var seededRand *rand.Rand = rand.New(rand.NewSource(time.Now().UnixNano()))
func StringWithCharset(length int, charset string) string {
	b := make([]byte, length)
	for i := range b {
    	b[i] = charset[seededRand.Intn(len(charset))]
	}
	return string(b)
}
func String(length int) string {
	return StringWithCharset(length, charset)
}

func CreateDB(web Website, dbAuth string, dbHost string) {
	dbSlice := strings.Split(dbHost, ":")
	db, err := sql.Open("mysql", fmt.Sprintf("%s@tcp(%s)/", dbAuth, dbHost))
	if err != nil {
		fmt.Println(err.Error())
	}
	_,err = db.Exec(fmt.Sprintf("CREATE DATABASE wp_%s", web.Username))
	if err != nil {
		fmt.Println(err.Error())
	}
	_,err = db.Exec(fmt.Sprintf("GRANT ALL ON wp_%s.* to %s@%s identified by '%s'", web.Username, web.Username, dbSlice[0], web.Password))
	if err != nil {
		fmt.Println(err.Error())
	}
}

func AddUser(web Website, dbAuth string, dbHost string) {
	db, err := sql.Open("mysql", fmt.Sprintf("%s@tcp(%s)/", dbAuth, dbHost))
	if err != nil {
		fmt.Println(err.Error())
	}
	_, err = db.Exec(fmt.Sprintf("INSERT INTO vsftpd.users (username, password, email, subdomain) VALUES ('%s',md5('%s'), '%s', '%s')", web.Username, web.Password, web.Email, web.Username))
	if err != nil {
		fmt.Println(err.Error())
	}
}

func SetupScript(web Website, dbHost string, dirPath string, confPath string) {
	newpath := filepath.Join(dirPath, web.Username)
	if _, err := os.Stat(newpath); os.IsNotExist(err) {
		os.Mkdir(newpath, os.ModePerm)
	}

	command := exec.Command("tar", "xzvf", "wordpress.zip", "--strip-components", "1", "-C", newpath)
	err := command.Run()
	if err != nil {
		fmt.Sprintf("cmd.Run() failed with %s\n", err)
	}

	err = os.Rename(newpath+"/wp-config-sample.php", newpath+"/wp-config.php")
	if err != nil {
		fmt.Println(err)
	}

	command = exec.Command("sed", "-i", "", fmt.Sprintf("s/nama_basis_data_di_sini/%s/g", "wp_"+web.Username), newpath+"/wp-config.php")
	err = command.Run()
	if err != nil {
		fmt.Sprintf("cmd.Run() failed with %s\n", err)
	}

	command = exec.Command("sed","-i", "", fmt.Sprintf("s/nama_pengguna_di_sini/%s/g", web.Username), newpath+"/wp-config.php")
	err = command.Run()
	if err != nil {
		fmt.Sprintf("cmd.Run() failed with %s\n", err)
	}

	command = exec.Command("sed", "-i", "", fmt.Sprintf("s/kata_sandi_di_sini/%s/g", web.Password), newpath+"/wp-config.php")
	err = command.Run()
	if err != nil {
		fmt.Sprintf("cmd.Run() failed with %s\n", err)
	}

	command = exec.Command("sed", "-i", "", fmt.Sprintf("s/localhost/%s/g", dbHost), newpath+"/wp-config.php")
	err = command.Run()
	if err != nil {
		fmt.Sprintf("cmd.Run() failed with %s\n", err)
	}

	command = exec.Command("cp", "./app.conf", filepath.Join(confPath, web.Username))
	err = command.Run()
	if err != nil {
		fmt.Sprintf("cmd.Run() failed with %s\n", err)
	}

	command = exec.Command("sed", "-i", "", fmt.Sprintf("s/USER_NAME/%s/g", web.Username), filepath.Join(confPath, web.Username))
	err = command.Run()
	if err != nil {
		fmt.Sprintf("cmd.Run() failed with %s\n", err)
	}
}

func SendMessage(conn *amqp.Connection, service string) {
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

	err = channel.Publish(
		"events", 	  // exchange
		"service", // routing key
		false, 		  // mandatory
		false,  	  // immediate
		amqp.Publishing{
			Body: []byte(service),
		},
	)

    if err != nil {
        panic("error publishing a message to the queue:" + err.Error())
    }
}

type Website struct {
    Username string
	Email  string
	SiteName string
	Password string
	SubDomain string
}