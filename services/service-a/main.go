package main

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"io/ioutil"
	"math/rand"
	"net/http"
	"net/http/httputil"
	"os"
	"strconv"
	"time"

	"github.com/janusky/pkg-commons/db"
	"github.com/janusky/pkg-commons/helper"

	runtime "github.com/banzaicloud/logrus-runtime-formatter"

	"github.com/google/uuid"
	"github.com/gorilla/mux"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"github.com/rs/cors"
	log "github.com/sirupsen/logrus"
	"github.com/streadway/amqp"
)

var (
	logLevel       = helper.GetEnv("LOG_LEVEL", "info")
	listenAddr     = helper.GetEnv("PORT", ":8080")
	serviceName    = helper.GetEnv("SERVICE_NAME", "Service service-a")
	message        = helper.GetEnv("GREETING", "Dummy, from service-a")
	allowedOrigins = helper.GetEnv("ALLOWED_ORIGINS", "*")
	queueName      = helper.GetEnv("QUEUE_NAME", "service-a.greeting")
	URLServiceB    = helper.GetEnv("SERVICE_B_INPUT_URL", "http://service-b:8080/api/greeting")
	rabbitMQConn   = helper.GetEnv("RABBITMQ_CONN", "amqp://guest:guest@rabbitmq:5672")
)

type Greeting struct {
	ID          string    `json:"id,omitempty"`
	ServiceName string    `json:"service,omitempty"`
	Message     string    `json:"message,omitempty"`
	CreatedAt   time.Time `json:"created,omitempty"`
	Hostname    string    `json:"hostname,omitempty"`
	Info        string    `json:"info,omitempty"`
}

const requestfinished = "request finished"

var (
	greetings []Greeting
	NotFound  = errors.New("Not found")
	rnd       = rand.New(rand.NewSource(time.Now().UnixNano()))
)

var connectBrokerSend *db.RabbitMQConection

func DummyHandler(w http.ResponseWriter, r *http.Request) {
	log.Info("service-a 1:HTTP Dummy")

	g := Greeting{
		ID:          uuid.New().String(),
		ServiceName: serviceName,
		Message:     message,
		CreatedAt:   time.Now().Local(),
		Hostname:    helper.GetHostname(),
		Info:        "service-a",
	}

	err := doGreeting(g, r)
	responder("greetings", w, r)(greetings, err)
}

func GreetingHandler(w http.ResponseWriter, r *http.Request) {
	log.Info("service-a 1:HTTP Greeting")

	var g Greeting

	err := helper.DecodeJSONBody(w, r, &g)
	if err != nil {
		var mr *helper.MalformedRequest
		if errors.As(err, &mr) {
			http.Error(w, mr.Msg, mr.Status)
		} else {
			log.Println(err.Error())
			http.Error(w, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
			// responder("greetings", w, r)(nil, err)
		}
		return
	}

	// Inicializa si es necesario
	if g.ID == "" {
		g.ID = uuid.New().String()
	}
	if g.Hostname == "" {
		g.Hostname = helper.GetHostname()
	}
	if g.Info == "" {
		g.Info = "service-a"
	}
	g.CreatedAt = time.Now().Local()

	err = doGreeting(g, r)
	responder("greetings", w, r)(greetings, err)
}

func doGreeting(greeting Greeting, r *http.Request) error {
	log.Debugf("request %v", r)
	greetings = nil
	greetings = append(greetings, greeting)

	callNextServiceWithTrace(URLServiceB, r)

	// Headers must be passed for Jaeger Distributed Tracing
	// https://istio.io/latest/docs/tasks/observability/distributed-tracing/overview/
	incomingHeaders := []string{
		"x-b3-flags",
		"x-b3-parentspanid",
		"x-b3-sampled",
		"x-b3-spanid",
		"x-b3-traceid",
		"x-ot-span-context",
		"x-request-id",
	}
	rabbitHeaders := amqp.Table{}
	for _, header := range incomingHeaders {
		if r.Header.Get(header) != "" {
			rabbitHeaders[header] = r.Header.Get(header)
		}
	}
	log.Debugf("headers %v", rabbitHeaders)

	body, err := json.Marshal(greeting)
	sendMessage(rabbitHeaders, body, connectBrokerSend.Connection)

	log.Debug(greetings)

	return err
}

func responder(resourceName string, w http.ResponseWriter, r *http.Request) func(data interface{}, err error) {
	slog := log.WithField("request", helper.NewRequestID())
	vlog := slog.WithFields(log.Fields{"uri": r.RequestURI, "method": r.Method, "remote": r.RemoteAddr, "resource": resourceName})
	vlog.Debug("request starting")
	return func(data interface{}, err error) {
		if err == NotFound {
			w.WriteHeader(http.StatusNotFound)
			vlog.WithError(err).WithFields(log.Fields{"status": http.StatusNotFound}).Info(requestfinished)
			return
		}
		if err != nil {
			log.Errorf("getting %s: %v", resourceName, err)
			w.WriteHeader(http.StatusInternalServerError)
			vlog.WithError(err).WithFields(log.Fields{"status": http.StatusInternalServerError}).Info(requestfinished)
			return
		}
		w.Header().Set("Content-Type", "application/json; charset=utf-8")
		w.WriteHeader(http.StatusOK)
		err = json.NewEncoder(w).Encode(data)
		if err != nil {
			log.Errorf("sending %s: %v", resourceName, err)
			slog.WithError(err).Info("writing response body")
		}
		vlog.WithFields(log.Fields{"status": http.StatusOK}).Info(requestfinished)
	}
}

func HealthCheckHandler(w http.ResponseWriter, _ *http.Request) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(http.StatusOK)
	_, err := w.Write([]byte("{\"alive\": true}"))
	if err != nil {
		log.Error(err)
	}
}

func healthReadiness() bool {
	//TODO 2021/10/01 manuelhernandez - Chequer servicio HTTP (service-b)

	if connectBrokerSend == nil || !connectBrokerSend.Active {
		log.WithError(errors.New("inactive to connect with rabbitmq SEND"))
		return false
	}
	return true
}

func HealthTestHandler(w http.ResponseWriter, _ *http.Request) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(http.StatusOK)

	test := healthReadiness()

	log.Debugf("service-a Health Test: %v", test)

	_, err := w.Write([]byte(fmt.Sprintf("{\"alive\": %v}", test)))
	if err != nil {
		log.Error(err)
	}
}

func ResponseStatusHandler(w http.ResponseWriter, r *http.Request) {
	params := mux.Vars(r)
	statusCode, err := strconv.Atoi(params["code"])
	if err != nil {
		log.Error(err)
	}
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(statusCode)
}

func RequestEchoHandler(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "text/plain; charset=utf-8")
	w.WriteHeader(http.StatusOK)

	requestDump, err := httputil.DumpRequest(r, true)
	if err != nil {
		log.Error(err)
	}
	_, err = fmt.Fprintf(w, string(requestDump))
	if err != nil {
		log.Error(err)
	}
}

func sendMessage(headers amqp.Table, body []byte, conn *amqp.Connection) {
	log.Infof("service-a 1-2:Send message Broker %s", queueName)
	log.Debugf("headers %v", headers)
	ch, err := conn.Channel()
	if err != nil {
		log.Error(err)
	}

	defer func(ch *amqp.Channel) {
		err := ch.Close()
		if err != nil {

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

	err = ch.Publish(
		"",
		q.Name,
		false,
		false,
		amqp.Publishing{
			Headers:     headers,
			ContentType: "application/json",
			Body:        body,
		})
	if err != nil {
		log.Error(err)
	}
}

func callNextServiceWithTrace(url string, r *http.Request) {
	log.Infof("service-a 1-1:Call %s", url)

	var tmpGreetings []Greeting

	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		log.Error(err)
	}

	incomingHeaders := []string{
		"x-b3-flags",
		"x-b3-parentspanid",
		"x-b3-sampled",
		"x-b3-spanid",
		"x-b3-traceid",
		"x-ot-span-context",
		"x-request-id",
	}

	for _, header := range incomingHeaders {
		if r.Header.Get(header) != "" {
			req.Header.Add(header, r.Header.Get(header))
		}
	}

	log.Debugf("call request %v", req)

	client := &http.Client{
		Timeout: time.Second * 10,
	}
	response, err := client.Do(req)
	if err != nil {
		log.Error(err)
	}

	defer func(Body io.ReadCloser) {
		err := Body.Close()
		if err != nil {
			log.Error(err)
		}
	}(response.Body)

	body, err := ioutil.ReadAll(response.Body)
	if err != nil {
		log.Error(err)
	}

	err = json.Unmarshal(body, &tmpGreetings)
	if err != nil {
		log.Error(err)
	}

	for _, r := range tmpGreetings {
		greetings = append(greetings, r)
	}
}

func newServer() (*http.Server, error) {
	c := cors.New(cors.Options{
		AllowedOrigins:   []string{allowedOrigins},
		AllowCredentials: true,
		AllowedMethods:   []string{"GET", "POST", "PUT", "DELETE", "PATCH", "OPTIONS", "HEAD"},
	})
	router := mux.NewRouter()
	api := router.PathPrefix("/api").Subrouter()
	api.HandleFunc("/dummy", DummyHandler).Methods("GET", "OPTIONS")
	api.HandleFunc("/greeting", GreetingHandler).Methods("POST")
	api.HandleFunc("/health", HealthCheckHandler).Methods("GET", "OPTIONS")
	api.HandleFunc("/test", HealthTestHandler).Methods("GET", "OPTIONS")
	api.HandleFunc("/request-echo", RequestEchoHandler).Methods(
		"GET", "POST", "PUT", "DELETE", "PATCH", "OPTIONS", "HEAD")
	api.HandleFunc("/status/{code}", ResponseStatusHandler).Methods("GET", "OPTIONS")
	api.Handle("/metrics", promhttp.Handler())
	handler := c.Handler(router)

	return &http.Server{
		Handler: handler,
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
		err := connectBrokerSend.Connection.Close()
		if err != nil {
			log.Error(err)
		}
		log.Debug("close connection RabbitMQ Send")
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
	// Connection RabbitMQ SEND
	go func() {
		reconnectDelay := 3 * time.Second
		retry := 10

		connectBrokerSend = db.NewRabbitMQConection(rabbitMQConn)

		err := helper.Retry(retry, reconnectDelay, func() error {
			if !connectBrokerSend.Active {
				return errors.New("unable to connect with rabbitmq SEND")
			}
			return nil
		})
		helper.AbortErr(err, "failed to connect with ConnectionBroker SEND")

		log.Info("connect with RabbitMQ SEND ok")
	}()
}
