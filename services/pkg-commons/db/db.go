package db

type Database interface {
	Close() error
	Ping() error
}
