package main

import (
	"fmt"
	"time"
)

type LogWriter struct{}

func (writer *LogWriter) Write(bytes []byte) (int, error) {
	timestamp := time.Now().UTC().Format("2006-01-02 15:04:05")
	return fmt.Printf("%v %v", timestamp, string(bytes))
}
