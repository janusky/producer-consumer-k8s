// TODO 09/09/2021 janusky@gmail.com - Change to https://www.ribice.ba/golang-rabbitmq-client/
// See:
// * https://qvault.io/golang/connecting-to-rabbitmq-in-golang/
// * https://levelup.gitconnected.com/creating-a-minimal-rabbitmq-client-using-go-cbcec1470950

package db

import (
	"log"
	"time"

	"github.com/streadway/amqp"
)

var (
	rabbitConn       *amqp.Connection
	rabbitCloseError chan *amqp.Error
)

type RabbitMQConection struct {
	Connection *amqp.Connection
	URI        string
	Active     bool
}

// Initialize new RabbitMQ connection
func NewRabbitMQConection(amqpUri string) *RabbitMQConection {
	var rabbitMQConn = &RabbitMQConection{
		URI: amqpUri,
	}

	// create the rabbitmq error channel
	rabbitCloseError = make(chan *amqp.Error)

	// run the callback in a separate thread
	go rabbitConnector(amqpUri, rabbitMQConn)

	// establish the rabbitmq connection by sending
	// an error and thus calling the error callback
	rabbitCloseError <- amqp.ErrClosed

	return rabbitMQConn
}

// Try to connect to the RabbitMQ server as long as it takes to establish a connection
func connectToRabbitMQ(uri string) *amqp.Connection {
	for {
		conn, err := amqp.Dial(uri)

		if err == nil {
			return conn
		}

		log.Println(err)
		log.Printf("Trying to reconnect to RabbitMQ at %s\n", uri)
		time.Sleep(500 * time.Millisecond)
	}
}

// re-establish the connection to RabbitMQ in case the connection has died
func rabbitConnector(amqpUri string, rabbitMQConection *RabbitMQConection) {
	var rabbitErr *amqp.Error

	for {
		rabbitErr = <-rabbitCloseError
		if rabbitErr != nil {
			// TODO 2021/09/27 janusky@gmail.com - Quitar o convertir a DEBUG
			log.Printf("Connecting .. to %s\n", amqpUri)

			rabbitConn = connectToRabbitMQ(amqpUri)
			rabbitCloseError = make(chan *amqp.Error)
			rabbitConn.NotifyClose(rabbitCloseError)

			// run your setup process here
			rabbitMQConection.Connection = rabbitConn
			rabbitMQConection.Active = true

			return
		}
	}
}
