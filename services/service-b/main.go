package main

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"os"
	"time"

	"github.com/janusky/pkg-commons/db"
	"github.com/janusky/pkg-commons/helper"

	runtime "github.com/banzaicloud/logrus-runtime-formatter"

	"github.com/google/uuid"
	"github.com/gorilla/mux"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	log "github.com/sirupsen/logrus"
	"github.com/streadway/amqp"
)

var (
	logLevel        = helper.GetEnv("LOG_LEVEL", "info")
	listenAddr      = helper.GetEnv("PORT", ":8080")
	serviceName     = helper.GetEnv("SERVICE_NAME", "Service service-b")
	message         = helper.GetEnv("GREETING", "Dummy, from Service service-b")
	queueName       = helper.GetEnv("QUEUE_NAME", "service-a.greeting")
	mongoURI        = helper.GetEnv("MONGO_CONN", "mongodb://mongodb:27017/admin")
	mongoUsername   = helper.GetEnv("MONGO_USERNAME", "")
	mongoPassword   = helper.GetEnv("MONGO_PASSWORD", "")
	mongoDatabase   = helper.GetEnv("MONGO_DATABASE", "service-b")
	mongoCollection = helper.GetEnv("MONGO_COLLECTION", "messages")
	rabbitMQConn    = helper.GetEnv("RABBITMQ_CONN", "amqp://guest:guest@rabbitmq:5672")
)

type Greeting struct {
	ID          string    `json:"id,omitempty"`
	ServiceName string    `json:"service,omitempty"`
	Message     string    `json:"message,omitempty"`
	CreatedAt   time.Time `json:"created,omitempty"`
	Hostname    string    `json:"hostname,omitempty"`
	Info        string    `json:"info,omitempty"`
}

var greetings []Greeting
var documentDB *db.DocumentDB
var connectBroker *db.RabbitMQConection

// *** HANDLERS ***

func DummyHandler(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(http.StatusOK)

	log.Info("service-b 1:HTTP Dummy")
	log.Debugf("request %v", r)

	greetings = nil

	tmpGreeting := Greeting{
		ID:          uuid.New().String(),
		ServiceName: serviceName,
		Message:     message,
		CreatedAt:   time.Now().Local(),
		Hostname:    helper.GetHostname(),
		Info:        "service-b",
	}

	greetings = append(greetings, tmpGreeting)

	err := doSave(tmpGreeting, documentDB)
	helper.AbortErr(err, "failed to Save in DocumentDB")

	err = json.NewEncoder(w).Encode(greetings)
	if err != nil {
		log.Error(err)
	}
	log.Debug(greetings)
}

func HealthCheckHandler(w http.ResponseWriter, _ *http.Request) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(http.StatusOK)
	_, err := w.Write([]byte("{\"alive\": true}"))
	if err != nil {
		log.Error(err)
	}
}

func HealthTestHandler(w http.ResponseWriter, _ *http.Request) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(http.StatusOK)

	test := healthReadiness()

	log.Debugf("fe-save Health Test: %v", test)

	_, err := w.Write([]byte(fmt.Sprintf("{\"alive\": %v}", test)))
	if err != nil {
		log.Error(err)
	}
}

// *** UTILITY FUNCTIONS ***

func healthReadiness() bool {
	var err error

	// FIXME 01/10/2021 manuelhernandez - Ver porque falla
	// err = documentDB.Ping()
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	err = documentDB.Client.Ping(ctx, nil)
	if err != nil {
		log.WithError(err).Error("failed to connect with DocumentDB")
		return false
	}

	if connectBroker == nil || !connectBroker.Active {
		log.WithError(errors.New("inactive to connect with rabbitmq"))
		return false
	}
	return true
}

func doSave(greeting Greeting, docDb *db.DocumentDB) error {
	log.Infof("service-b 1-1:Save Database ID:%s", greeting.ID)

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	// collection := client.Database(mongoDatabase).Collection(mongoCollection)
	// _, err = collection.InsertOne(ctx, greeting)
	greeting.Info = "Finish"
	_, err := docDb.InsertOne(ctx, mongoDatabase, mongoCollection, greeting)

	return err
}

func getMessages(conn *amqp.Connection) {
	log.Debugf("get messages %s", queueName)

	ch, err := conn.Channel()
	if err != nil {
		log.Error(err)
	}
	defer func(ch *amqp.Channel) {
		err := ch.Close()
		if err != nil {
			log.Error(err)
		}
	}(ch)

	q, err := ch.QueueDeclare(
		queueName,
		false,
		false,
		false,
		false,
		nil,
	)
	if err != nil {
		log.Error(err)
	}

	msgs, err := ch.Consume(
		q.Name,
		"service-b",
		true,
		false,
		false,
		false,
		nil,
	)
	if err != nil {
		log.Error(err)
	}

	forever := make(chan bool)

	go func() {
		for delivery := range msgs {
			data := deserialize(delivery.Body)
			log.Infof("service-b 1:Consume %s", data.ID)
			log.Debugf("read %v", data)

			err := doSave(data, documentDB)
			helper.AbortErr(err, "failed to Save in DocumentDB")
		}
	}()

	<-forever
}

func deserialize(b []byte) (t Greeting) {
	var tmpGreeting Greeting
	buf := bytes.NewBuffer(b)
	decoder := json.NewDecoder(buf)
	err := decoder.Decode(&tmpGreeting)
	if err != nil {
		log.Error(err)
	}
	return tmpGreeting
}

func newServer() (*http.Server, error) {
	router := mux.NewRouter()
	api := router.PathPrefix("/api").Subrouter()
	api.HandleFunc("/greeting", DummyHandler).Methods("GET")
	api.HandleFunc("/health", HealthCheckHandler).Methods("GET", "OPTIONS")
	api.HandleFunc("/test", HealthTestHandler).Methods("GET", "OPTIONS")
	api.Handle("/metrics", promhttp.Handler())

	return &http.Server{
		Handler: router,
		Addr:    listenAddr,
	}, nil
}

func init() {
	formatter := runtime.Formatter{ChildFormatter: &log.JSONFormatter{}}
	formatter.Line = true
	log.SetFormatter(&formatter)
	log.SetOutput(os.Stdout)
	level, err := log.ParseLevel(logLevel)
	if err != nil {
		log.Error(err)
	}
	log.SetLevel(level)
}

func main() {
	go initConnections()

	defer func() {
		err := connectBroker.Connection.Close()
		if err != nil {
			log.Error(err)
		}
		log.Debug("close connection Broker Messages")
	}()

	defer func() {
		err := documentDB.Close()
		if err != nil {
			log.Warn(err)
		}
		log.Debug("close connection DocumentDB")
	}()

	server, err := newServer()
	helper.AbortErr(err, "creating http server")

	done := helper.HandleExit(func() {
		ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
		defer cancel()
		log.Info("starting http server shut down")
		if err := server.Shutdown(ctx); err != nil {
			log.WithError(err).Error("shutting down http server")
		}
	})

	err = server.ListenAndServe()
	if err != http.ErrServerClosed {
		log.WithError(err).Error("http server exited")
		os.Exit(1)
	}
	log.Info("http server shut down")

	<-done
	log.Info("all done")
}

func initConnections() {
	// Connection RabbitMQ GET
	go func() {
		reconnectDelay := 3 * time.Second
		retry := 10

		connectBroker = db.NewRabbitMQConection(rabbitMQConn)

		err := helper.Retry(retry, reconnectDelay, func() error {
			if !connectBroker.Active {
				return errors.New("unable to connect with rabbitmq")
			}
			return nil
		})
		helper.AbortErr(err, "failed to connect with ConnectionBroker")

		log.Info("connect with RabbitMQ GET ok")
		getMessages(connectBroker.Connection)
	}()

	// Connection MongoDB
	go func() {
		var err error

		documentDB, err = helper.ConnectionDocument(mongoURI, mongoUsername, mongoPassword)
		helper.AbortErr(err, "failed to connect with DocumentDB")
		log.Info("connect with MongoDB ok")
	}()
}
