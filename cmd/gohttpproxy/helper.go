package main

import (
	"fmt"
	"log"
	"net"
	"os"
	"time"
)

// appendToFile appends plain text to a log file
func appendToFile(logFileName string, logEntry string) {
	f, err := os.OpenFile(logFileName, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
	if err != nil {
		log.Fatal("Error opening log file:", err)
	}
	defer f.Close()

	if _, err := f.WriteString(logEntry); err != nil {
		log.Fatal("Error writing to log file:", err)
	}
}

// Validate the port before establishing it.
func validatePort(host, port string, proxyAddr bool) bool {
	if len(host) == 0 || len(port) == 0 {
		log.Fatal("Error empty value, set Host: %s Port: %s", host, port)
	}

	// only validate port use if the proxy the local port
	if proxyAddr {
		serverAddress := fmt.Sprintf("%s:%s", host, port)
		conn, err := net.DialTimeout("tcp", serverAddress, 2*time.Second)
		if err == nil {
			_ = conn.Close()
			log.Fatalf("Socket is in use!")
			return false
		}
	}

	// If the connection failed with an error, it's usually a sign that the port is available
	return true
}
