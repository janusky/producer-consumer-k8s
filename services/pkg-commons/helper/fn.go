package helper

import (
	"fmt"
	"log"
	"os"
	"os/signal"
	"syscall"
	"time"
)

func Retry(attempts int, sleep time.Duration, f func() error) (err error) {
	for i := 0; i < attempts; i++ {
		if i > 0 {
			log.Println("retrying after error:", err)
			time.Sleep(sleep)
			sleep *= 2
		}
		err = f()
		if err == nil {
			return nil
		}
	}
	return fmt.Errorf("after %d attempts, last error: %s", attempts, err)
}

func HandleExit(handler func()) chan struct{} {
	done := make(chan struct{}, 1)
	go func() {
		quit := make(chan os.Signal)
		// signal.Notify(quit, os.Interrupt, os.Kill)
		signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM, syscall.SIGKILL)
		<-quit
		log.Println("received interrupt signal")
		handler()
		close(done)
	}()
	return done
}
