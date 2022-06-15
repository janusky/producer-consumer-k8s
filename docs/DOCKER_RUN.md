# producer-consumer-k8s

Ejecutar modo simple con **docker-compose**.

Architecture Docker Images

* service-a - build [/services/service-a](/services/service-a/main.go) or [janusky/service-a](https://hub.docker.com/repository/docker/janusky/service-a)
* service-b - build [/services/service-b](/services/service-b/main.go) or [janusky/service-b](https://hub.docker.com/repository/docker/janusky/service-b)
* rabbitmq - [rabbitmq:3.8.16-management-alpine](https://hub.docker.com/_/rabbitmq)
* mongodb - [mongo:5.0.2](https://hub.docker.com/_/mongo)
* mongo-express - [mongo-express:1.0.0-alpha.4](https://hub.docker.com/_/mongo-express)

>NOTA: Si en su red tiene problemas para descargar la librerías utilizadas en la construcción de las imágenes de los servicios debe reintentar hasta lograrlo.

## Run Docker

Ejecución por defecto implicando lo modelado en la arquitectura.


Optar por una opción (default or development)

```sh
# DEFAULT Option
# Está ejecución requiere de la existencia de imágenes en dockerhub
docker-compose up -d

#====================================================#

# DEVELOPMENT Option
# Ejecutar realizando la contrucción local de imagenes para los servicios (producer-consumer-k8s/services)
docker-compose -f docker-compose-dev.yaml up -d
```

Realizar ingreso de datos

* http://localhost:8080/api/request-echo

```sh
curl -v http://localhost:8080/api/dummy

curl -v --noproxy '*' -XPOST http://localhost:8080/api/greeting -H 'Content-Type: application/json' -d "{
  \"id\": \"$(uuidgen)\",
  \"service\": \"Service service-uuidgen\",
  \"message\": \"Dummy, from service-uuidgen\"
}"

curl -v --noproxy '*' -XPOST http://localhost:8080/api/greeting -H 'Content-Type: application/json' -d '{
  "service": "Service service-z",
  "message": "Dummy, from service-z"
}'
```

Chequear Mongo DB

* http://localhost:8081/

Chequear RabbitMQ

* http://localhost:15672/ (guest/guest)
