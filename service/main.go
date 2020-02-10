package main

import (
	"fmt"
	"flag"
	"os"
	"strings"
	"os/exec"
	"encoding/base64"
	"path/filepath"
	"encoding/json"
	"github.com/streadway/amqp"
	"net/smtp"
	"net/http"
	"net/url"
	"time"
	"io/ioutil"
)

func main() {
	rabbitmqPtr := flag.String("rabbitmq", "guest:guest@localhost:5672", "rabbit mq connection string (guest:guest@localhost:5672)")
	smtpPtr := flag.String("smtp", "192.168.56.5:25", "smtp host")
	stackNamePtr := flag.String("stack-name", "shared", "docker stack name")
	dirPathPtr := flag.String("file-dir", "./", "wordpress root path")
	confPathPtr := flag.String("conf-dir", "./", "nginx conf path")
	flag.Parse()
    ReceiveMessage(*rabbitmqPtr, *stackNamePtr, *dirPathPtr, *confPathPtr, *smtpPtr)
}

func ReceiveMessage(r string, stack string, dirPath string, confPath string, smtpAddr string) {
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
			if (website.Action == "install") {
				err := SetupScript(website, dirPath, confPath)
				if err != nil {
					fmt.Println(err.Error())
				}

				RestartNginx(conn, stack+"_nginx")
				time.Sleep(40 * time.Second)

				err = WordpressRegistration(website)
				if err != nil {
					fmt.Println(err.Error())
				}

				err = SendMail(smtpAddr, website.Email, "Registrasi Web Berhasil", fmt.Sprintf("Registrasi web berhasil<br />Username: %s<br />Password: %s", website.Username, website.Password))
				if err != nil {
					fmt.Println(err.Error())
				}
			} else if (website.Action == "update_domain") {
				err := UpdateDomain(website, confPath)
				if err != nil {
					fmt.Println(err.Error())
				}

				err = SendMail(smtpAddr, website.Email, "Perubahan Subdomain Berhasil", fmt.Sprintf("Perubahan subdomain ke %s berhasil", website.SubDomain))
				if err != nil {
					fmt.Println(err.Error())
				}
				RestartNginx(conn, stack+"_nginx")
			} else if (website.Action == "reset_password") {
				err := SendMail(smtpAddr, website.Email, "Password baru anda", fmt.Sprintf("Password baru anda adalah %s", website.Password))
				if err != nil {
					fmt.Println(err.Error())
				}
			} else if (website.Action == "delete") {
				Uninstall(website, dirPath, confPath)
				RestartNginx(conn, stack+"_nginx")
			}
			m.Ack(false)
		}
	}()
	fmt.Println(" [*] Waiting for messages. To exit press CTRL+C")
	<-forever
}

func WordpressRegistration(website Website) error {
	repeat := 0
	client := http.Client{
		Timeout: 60 * time.Second,
	}
	for {
		resp, err := client.Get(fmt.Sprintf("http://%s.website.ku", website.Username))
		if err != nil {
			return err
		}

		body, err := ioutil.ReadAll(resp.Body)
		if err != nil {
			return err
		}

		fmt.Sprintf("Resp: %s, Retry: %v", string(body), repeat)
		if ((string(body) != "Available") || repeat > 10) {
			break
		}

		repeat++
		time.Sleep(time.Duration(repeat) * time.Second * 6)
	}
	data := url.Values{}
	data.Set("admin_email", website.Email)
	data.Set("admin_password", website.Password)
	data.Set("admin_password2", website.Password)
	data.Set("user_name", website.Username)
	data.Set("weblog_title", website.Username)
	data.Set("Submit", "Install+WordPress")

	r, _ := http.NewRequest("POST", fmt.Sprintf("http://%s.website.ku/wp-admin/install.php?step=2", website.Username), strings.NewReader(data.Encode()))
	r.Header.Add("Content-Type", "application/x-www-form-urlencoded")

	resp, _ := client.Do(r)
	fmt.Println(resp.Status)
	return nil
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

func SetupScript(web Website, dirPath string, confPath string) error {
	newpath := filepath.Join(dirPath, web.Username)
	if _, err := os.Stat(newpath); os.IsNotExist(err) {
		os.Mkdir(newpath, os.ModePerm)
	}

	fmt.Println("UNPACKING WP SCRIPT")
	command := exec.Command("tar", "--same-owner", "-xzf", dirPath+"/wordpress.tar.gz", "-C", newpath)
	err := command.Run()
	if err != nil {
		return err
	}

	fmt.Println("SETTING WP CONFIG")

	command = exec.Command("sed", "-i", fmt.Sprintf("s/nama_basis_data_di_sini/%s/g", "wp_"+web.Username), newpath+"/wp-config.php")
	err = command.Run()
	if err != nil {
		return err
	}

	command = exec.Command("sed","-i", fmt.Sprintf("s/nama_pengguna_di_sini/%s/g", web.Username), newpath+"/wp-config.php")
	err = command.Run()
	if err != nil {
		return err
	}

	command = exec.Command("sed", "-i", fmt.Sprintf("s/kata_sandi_di_sini/%s/g", web.Password), newpath+"/wp-config.php")
	err = command.Run()
	if err != nil {
		return err
	}

	command = exec.Command("sed", "-i", fmt.Sprintf("s/wp_cache_salt/%s/g", web.Username), newpath+"/wp-config.php")
	err = command.Run()
	if err != nil {
		return err
	}

	command = exec.Command("sed", "-i", "s/localhost/192.168.56.5:3306/g", newpath+"/wp-config.php")
	err = command.Run()
	if err != nil {
		return err
	}

	command = exec.Command("chown", "-R", "1000:1000", newpath)
	err = command.Run()
	if err != nil {
		return err
	}

	fmt.Println("SETTING NGINX SCRIPT")
	f, err := os.Create(fmt.Sprintf("%s/%s.conf", confPath, web.Username))
    if err != nil {
        return err
    }
    conf := `server {
        listen       80;
        server_name  `+web.Username + `.website.ku;

        # note that these lines are originally from the "location /" block
        root   /opt/app/`+web.Username+`;
        index index.php index.html index.htm;

        location ~ \.php$ {
            resolver 127.0.0.11 ipv6=off;
            try_files $uri =404;
            set $upstream "php-fpm:9000";
            fastcgi_pass $upstream;
            fastcgi_index index.php;
            fastcgi_param SCRIPT_FILENAME $document_root$fastcgi_script_name;
            fastcgi_read_timeout 300;
            include fastcgi_params;
        }

        location ~ \.(ogg|ogv|svg|svgz|eot|otf|woff|mp4|ttf|css|rss|atom|js|jpg|jpeg|gif|png|ico|zip|tgz|gz|rar|bz2|doc|xls|exe|ppt|tar|mid|midi|wav|bmp|rtf)$ {
            expires max;
            log_not_found off;
            access_log off;
        }
    }`
    _, err = f.WriteString(conf)
    if err != nil {
        f.Close()
        return err
    }
    return nil
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

func UpdateDomain(web Website, confPath string) error {
	command := exec.Command("sed", "-i", fmt.Sprintf("s/%s/%s/g", web.CurrentSubDomain, web.SubDomain), fmt.Sprintf("%s/%s.conf", confPath, web.Username))
	err := command.Run()
	if err != nil {
		return err
	}
	return nil
}

func Uninstall(web Website, dirPath string, confPath string) error {
	command := exec.Command("rm", "-rf", filepath.Join(dirPath, web.Username))
	err := command.Run()
	if err != nil {
		return err
	}

	command = exec.Command("rm", "-f", fmt.Sprintf("%s/%s.conf", confPath, web.Username))
	err = command.Run()
	if err != nil {
		return err
	}
	return nil
}

type Website struct {
    Username string
	Email  string
	Password string
	CurrentSubDomain string
	SubDomain string
	Action string
}