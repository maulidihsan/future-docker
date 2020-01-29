package main

import (
	"fmt"
	"flag"
	"os"
	"os/exec"
	"math/rand"
	"time"
	"encoding/base64"
	"path/filepath"
	"database/sql"
	"encoding/json"
	_ "github.com/go-sql-driver/mysql"
	"github.com/streadway/amqp"
	"net/smtp"
)

func main() {
	rabbitmqPtr := flag.String("rabbitmq", "guest:guest@localhost:5672", "rabbit mq connection string (guest:guest@localhost:5672)")
	smtpPtr := flag.String("smtp", "192.168.56.5:25", "smtp host")
	stackNamePtr := flag.String("stack-name", "shared", "docker stack name")
	dbauthPtr := flag.String("mysql-auth", "root:password", "msyql username:password")
	dbhostPtr := flag.String("mysql-host", "127.0.0.1:3006", "mysql host (127.0.0.1:3306) root:password@tcp(127.0.0.1:3306)")
	dirPathPtr := flag.String("file-dir", "./", "wordpress root path")
	confPathPtr := flag.String("conf-dir", "./", "nginx conf path")
	flag.Parse()
    ReceiveMessage(*rabbitmqPtr, *stackNamePtr, *dbauthPtr, *dbhostPtr, *dirPathPtr, *confPathPtr, *smtpPtr)
}

func ReceiveMessage(r string, stack string, dbAuth string, dbHost string, dirPath string, confPath string, smtpAddr string) {
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
		"web_events", // routing key
		"events", // exchange name
		false,
		nil,
	)

    if err != nil {
        panic("error binding to the queue: " + err.Error())
    }

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
			fmt.Println(website)
			if (website.Action == "create") {
				website.Password = String(12)
				CreateDB(website, dbAuth, dbHost)
				AddUser(website, dbAuth, dbHost)
				SetupScript(website, dirPath, confPath)
				err := SendMail(smtpAddr, website.Email, "Registrasi Web Berhasil", fmt.Sprintf("Registrasi web berhasil<br />Username:%s<br />Password:%s", website.Username, website.Password))
				if err != nil {
					fmt.Println(err.Error())
				}
			} else if (website.Action == "update") {
				UpdateWeb(website, dbAuth, dbHost, confPath)
				err := SendMail(smtpAddr, website.Email, "Perubahan Subdomain Berhasil", fmt.Sprintf("Perubahan subdomain ke %s berhasil", website.SubDomain))
				if err != nil {
					fmt.Println(err.Error())
				}
			} else if (website.Action == "delete") {
				RemoveWeb(website, dbAuth, dbHost, dirPath, confPath)
			}
			RestartNginx(conn, stack+"_nginx")
			m.Ack(false)
		}
	}()
	fmt.Printf(" [*] Waiting for messages. To exit press CTRL+C")
	<-forever
}

func SendMail(addr string, to string, subject string, message string) error {
	fmt.Println("SENDING MAIL")
	from := "root@mail.website.ku"
	c, err := smtp.Dial(fmt.Sprintf("%s", addr))
	if err != nil {
			fmt.Println(err.Error())
	}
	defer c.Close()
	c.Mail(from)
	c.Rcpt(to)

	wc, err := c.Data()
	if err != nil {
			fmt.Println(err.Error())
	}
	body := "To: " + to + "\r\n" +
			"From: " + from + "\r\n" +
			"Subject: " + subject + "\r\n" +
			"Content-Type: text/html; charset=\"UTF-8\"\r\n" +
			"Content-Transfer-Encoding: base64\r\n" +
			"\r\n" + base64.StdEncoding.EncodeToString([]byte(message))
	_, err = wc.Write([]byte(body))
	if err != nil {
			return err
	}
	err = wc.Close()
	if err != nil {
			return err
	}
	return c.Quit()
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
	fmt.Println("CREATING DB")
	db, err := sql.Open("mysql", fmt.Sprintf("%s@tcp(%s)/", dbAuth, dbHost))
	if err != nil {
		fmt.Println(err.Error())
	}
	_,err = db.Exec(fmt.Sprintf("CREATE DATABASE wp_%s", web.Username))
	if err != nil {
		fmt.Println(err.Error())
	}
	_,err = db.Exec(fmt.Sprintf("GRANT ALL ON wp_%s.* to %s@'%%' identified by '%s'", web.Username, web.Username, web.Password))
	if err != nil {
		fmt.Println(err.Error())
	}
}

func AddUser(web Website, dbAuth string, dbHost string) {
	fmt.Println("ADDING FTP USER")
	db, err := sql.Open("mysql", fmt.Sprintf("%s@tcp(%s)/", dbAuth, dbHost))
	if err != nil {
		fmt.Println(err.Error())
	}
	_, err = db.Exec(fmt.Sprintf("INSERT INTO vsftpd.users (username, password, email, subdomain) VALUES ('%s',md5('%s'), '%s', '%s')", web.Username, web.Password, web.Email, web.Username))
	if err != nil {
		fmt.Println(err.Error())
	}
}

func SetupScript(web Website, dirPath string, confPath string) {
	newpath := filepath.Join(dirPath, web.Username)
	if _, err := os.Stat(newpath); os.IsNotExist(err) {
		os.Mkdir(newpath, os.ModePerm)
	}

	fmt.Println("UNPACKING WP SCRIPT")
	command := exec.Command("tar", "--same-owner", "xzf", dirPath+"/wordpress.tar.gz", "-C", newpath)
	err := command.Run()
	if err != nil {
		fmt.Sprintf("cmd.Run() failed with %s\n", err)
	}

	fmt.Println("SETTING WP CONFIG")

	command = exec.Command("sed", "-i", fmt.Sprintf("s/nama_basis_data_di_sini/%s/g", "wp_"+web.Username), newpath+"/wp-config.php")
	err = command.Run()
	if err != nil {
		fmt.Sprintf("cmd.Run() failed with %s\n", err)
	}

	command = exec.Command("sed","-i", fmt.Sprintf("s/nama_pengguna_di_sini/%s/g", web.Username), newpath+"/wp-config.php")
	err = command.Run()
	if err != nil {
		fmt.Sprintf("cmd.Run() failed with %s\n", err)
	}

	command = exec.Command("sed", "-i", fmt.Sprintf("s/kata_sandi_di_sini/%s/g", web.Password), newpath+"/wp-config.php")
	err = command.Run()
	if err != nil {
		fmt.Sprintf("cmd.Run() failed with %s\n", err)
	}

	command = exec.Command("sed", "-i", fmt.Sprintf("s/wp_cache_salt/%s/g", web.Username), newpath+"/wp-config.php")
	err = command.Run()
	if err != nil {
		fmt.Sprintf("cmd.Run() failed with %s\n", err)
	}

	command = exec.Command("sed", "-i", "s/localhost/192.168.56.5:3306/g", newpath+"/wp-config.php")
	err = command.Run()
	if err != nil {
		fmt.Sprintf("cmd.Run() failed with %s\n", err)
	}

	command = exec.Command("chown", "-R", "1000:1000", newpath)
	err = command.Run()
	if err != nil {
		fmt.Sprintf("cmd.Run() failed with %s\n", err)
	}

	fmt.Println("SETTING NGINX SCRIPT")
	f, err := os.Create(fmt.Sprintf("%s/%s.conf", confPath, web.Username))
    if err != nil {
        fmt.Println(err)
        return
    }
    conf := `server {
        listen       80;
        server_name  `+web.Username + `.website.ku;

        # note that these lines are originally from the "location /" block
        root   /opt/app/`+web.Username+`;
        index index.php index.html index.htm;

        location ~ \.php$ {
            resolver 127.0.0.11 ipv6=off;
            set $phpFPMHost "php-fpm:9000";
            try_files $uri =404;
            fastcgi_pass $phpFPMHost;
            fastcgi_index index.php;
            fastcgi_param SCRIPT_FILENAME $document_root$fastcgi_script_name;
            include fastcgi_params;
        }
    }`
    _, err = f.WriteString(conf)
    if err != nil {
        fmt.Println(err)
        f.Close()
        return
    }
}

func RestartNginx(conn *amqp.Connection, service string) {
    channel, err := conn.Channel()
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

    err = channel.Publish(
		"events",         // exchange
		"service_events", // routing key
		false,            // mandatory
		false,            // immediate
		amqp.Publishing{
			Body: []byte(service),
		},
    )

    if err != nil {
        panic("error publishing a message to the queue:" + err.Error())
    }
}

func UpdateWeb(web Website, dbAuth string, dbHost string, confPath string) {
	db, err := sql.Open("mysql", fmt.Sprintf("%s@tcp(%s)/", dbAuth, dbHost))
	if err != nil {
		fmt.Println(err.Error())
	}
	var subdomain string
	row := db.QueryRow(`SELECT subdomain FROM vsftpd.users WHERE username='$1';`, web.Username)
	err = row.Scan(&subdomain)
	if err != nil {
		fmt.Println(err.Error())
	}
	_, err = db.Exec(fmt.Sprintf("UPDATE vsftpd.users SET subdomain='%s' WHERE username='%s'", web.SubDomain, web.Username))
	if err != nil {
		fmt.Println(err.Error())
	}
	command := exec.Command("sed", "-i", fmt.Sprintf("s/%s/%s/g", subdomain, web.SubDomain), fmt.Sprintf("%s/%s.conf", confPath, web.Username))
	err = command.Run()
	if err != nil {
		fmt.Sprintf("cmd.Run() failed with %s\n", err)
	}
}

func RemoveWeb(web Website, dbAuth string, dbHost string, dirPath string, confPath string) {
	db, err := sql.Open("mysql", fmt.Sprintf("%s@tcp(%s)/", dbAuth, dbHost))
	if err != nil {
		fmt.Println(err.Error())
	}
	_, err = db.Exec(fmt.Sprintf("DELETE FROM vsftpd.users WHERE WHERE username='%s'", web.Username))
	if err != nil {
		fmt.Println(err.Error())
	}
	command := exec.Command("rm", "-rf", filepath.Join(dirPath, web.Username))
	err = command.Run()
	if err != nil {
		fmt.Sprintf("cmd.Run() failed with %s\n", err)
	}

	command = exec.Command("rm", "-f", fmt.Sprintf("%s/%s.conf", dirPath, web.Username))
	err = command.Run()
	if err != nil {
		fmt.Sprintf("cmd.Run() failed with %s\n", err)
	}
}

type Website struct {
    Username string
	Email  string
	Password string
	SubDomain string
	Action string
}