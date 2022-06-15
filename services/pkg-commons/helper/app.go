package helper

import (
	"context"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log"
	"math/rand"
	"net/http"
	"os"
	"strings"
	"time"

	"github.com/golang/gddo/httputil/header"

	"github.com/janusky/pkg-commons/db"
)

const (
	Attempts       int           = 10
	ReconnectDelay time.Duration = 3 * time.Second
)

var rnd = rand.New(rand.NewSource(time.Now().UnixNano()))

func GetEnv(key, fallback string) string {
	if value, ok := os.LookupEnv(key); ok {
		return value
	}
	return fallback
}

func GetHostname() string {
	hostname, err := os.Hostname()
	if err != nil {
		log.Println(err)
	}
	return hostname
}

func AbortErr(err error, msg string) {
	if err != nil {
		Abortef(err, msg)
	}
}

func Abortef(err error, msg string, args ...interface{}) {
	Abortf(msg+": %v", append(args, err))
}

func Abortf(msg string, args ...interface{}) {
	s := fmt.Sprintf(msg, args...)
	log.Println("received interrupt signal")
	log.Printf("error: %s", s)
	os.Exit(1)
}

type MalformedRequest struct {
	Status int
	Msg    string
}

func (mr *MalformedRequest) Error() string {
	return mr.Msg
}

func DecodeJSONBody(w http.ResponseWriter, r *http.Request, dst interface{}) error {
	if r.Header.Get("Content-Type") != "" {
		value, _ := header.ParseValueAndParams(r.Header, "Content-Type")
		if value != "application/json" {
			msg := "Content-Type header is not application/json"
			return &MalformedRequest{Status: http.StatusUnsupportedMediaType, Msg: msg}
		}
	}

	r.Body = http.MaxBytesReader(w, r.Body, 1048576)

	dec := json.NewDecoder(r.Body)
	dec.DisallowUnknownFields()

	err := dec.Decode(&dst)
	if err != nil {
		var syntaxError *json.SyntaxError
		var unmarshalTypeError *json.UnmarshalTypeError

		switch {
		case errors.As(err, &syntaxError):
			msg := fmt.Sprintf("Request body contains badly-formed JSON (at position %d)", syntaxError.Offset)
			return &MalformedRequest{Status: http.StatusBadRequest, Msg: msg}

		case errors.Is(err, io.ErrUnexpectedEOF):
			msg := fmt.Sprintf("Request body contains badly-formed JSON")
			return &MalformedRequest{Status: http.StatusBadRequest, Msg: msg}

		case errors.As(err, &unmarshalTypeError):
			msg := fmt.Sprintf("Request body contains an invalid value for the %q field (at position %d)", unmarshalTypeError.Field, unmarshalTypeError.Offset)
			return &MalformedRequest{Status: http.StatusBadRequest, Msg: msg}

		case strings.HasPrefix(err.Error(), "json: unknown field "):
			fieldName := strings.TrimPrefix(err.Error(), "json: unknown field ")
			msg := fmt.Sprintf("Request body contains unknown field %s", fieldName)
			return &MalformedRequest{Status: http.StatusBadRequest, Msg: msg}

		case errors.Is(err, io.EOF):
			msg := "Request body must not be empty"
			return &MalformedRequest{Status: http.StatusBadRequest, Msg: msg}

		case err.Error() == "http: request body too large":
			msg := "Request body must not be larger than 1MB"
			return &MalformedRequest{Status: http.StatusRequestEntityTooLarge, Msg: msg}

		default:
			return err
		}
	}

	err = dec.Decode(&struct{}{})
	if err != io.EOF {
		msg := "Request body must only contain a single JSON object"
		return &MalformedRequest{Status: http.StatusBadRequest, Msg: msg}
	}

	return nil
}

func NewRequestID() string {
	bs := make([]byte, 4)
	rnd.Read(bs)
	return hex.EncodeToString(bs)
}

//TODO 2021/09/27 janusky@gmail.com - Asegurarse que está funcionando bien db.NewDocumentDB..
func ConnectionDocument(mongoURI, mongoUsername, mongoPassword string) (*db.DocumentDB, error) {
	var docDB *db.DocumentDB
	var err error

	ctx, _ := context.WithTimeout(context.Background(), 10*time.Second)

	err = Retry(Attempts, ReconnectDelay, func() error {
		docDB, err = db.NewDocumentDB(ctx, mongoURI, mongoUsername, mongoPassword)
		return err
	})
	return docDB, err
}
