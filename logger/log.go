package logger

import "log"

type myLogger struct {
}

func init() {
	log.Default()
}
