# build based on:
# https://medium.com/@chemidy/create-the-smallest-and-secured-golang-docker-image-based-on-scratch-4752223b7324
# date: 2021-05-29

################################
# STEP 1 build executable binary
################################
FROM golang:1.16.4-alpine3.13 AS builder

EXPOSE 8080

# Application Path to build
ARG APP_PATH

# Install git, zoneinfo, and SSL certs
RUN apk update && apk add --no-cache git ca-certificates tzdata

# Create unprivileged appuser
RUN adduser -D -g '' appuser

# Copy file(s)
WORKDIR /go/src/app
COPY $APP_PATH/main.go .

# Paquete con funcionalidades
COPY ./pkg-commons /go/src/github.com/janusky/pkg-commons

# Fix alternativo (evaluar su utilización)
# http.postBuffer: 1048576000 | 524288000 | 1024M
RUN git config --global http.sslverify false
RUN git config --global http.postBuffer 1024M
RUN git config --global https.postBuffer 1024M
ENV GOPROXY=direct

# creates go.mod file to track code dependencies
RUN go mod init

RUN go mod edit -replace github.com/janusky/pkg-commons=/go/src/github.com/janusky/pkg-commons

# ensures go.mod file matches source code in the module
RUN go mod tidy

# Disable crosscompiling
ENV CGO_ENABLED=0

# Compile Linux only
ENV GOOS=linux

# Build the binary - remove debug info and compile only for linux target
RUN go build -ldflags '-w -s' -a -installsuffix cgo -o /go/bin/app .

############################
# STEP 2 build a small image
############################
FROM scratch

LABEL maintainer="Manuel A. Hernandez <janusky@gmail.com>"
ENV REFRESHED_AT 2021-08-25

# Import the user and group files from the builder
COPY --from=builder /etc/passwd /etc/passwd

# Import the zoneinfo and SSL cert files from the builder
COPY --from=builder /usr/share/zoneinfo /usr/share/zoneinfo
COPY --from=builder /etc/ssl/certs/ca-certificates.crt /etc/ssl/certs/

# Copy static executable
COPY --from=builder /go/bin/app /go/bin/app

# Use an unprivileged user
USER appuser

# Run the app binary
ENTRYPOINT ["/go/bin/app"]
