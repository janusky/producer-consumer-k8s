module github.com/janusky/sevice-a

go 1.16

replace github.com/janusky/pkg-commons => ../pkg-commons

require (
	github.com/banzaicloud/logrus-runtime-formatter v0.0.0-20190729070250-5ae5475bae5e
	github.com/golang/gddo v0.0.0-20210115222349-20d68f94ee1f
	github.com/google/uuid v1.2.0
	github.com/gorilla/mux v1.8.0
	github.com/prometheus/client_golang v1.10.0
	github.com/rs/cors v1.7.0
	github.com/sirupsen/logrus v1.8.1
	github.com/streadway/amqp v1.0.0
)
