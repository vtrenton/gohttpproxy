package main

import (
	"bytes"
	"fmt"
	"io"
	"net/http"
)

// logRequest logs the incoming request headers and body in plain text
func logRequest(logFileName string, r *http.Request) {
	logEntry := fmt.Sprintf("Incoming Request: %v %v %v\n", r.Method, r.URL, r.Proto)

	// Log headers
	for name, values := range r.Header {
		for _, value := range values {
			logEntry += fmt.Sprintf("Header: %v: %v\n", name, value)
		}
	}

	// Log body
	if r.Body != nil {
		bodyBytes, _ := io.ReadAll(r.Body)
		r.Body = io.NopCloser(bytes.NewBuffer(bodyBytes)) // Reassign body for reuse
		logEntry += fmt.Sprintf("Body: %s\n", string(bodyBytes))
	}

	// Write the log entry to file
	logEntry += "\n"
	fileMutex.Lock()
	defer fileMutex.Unlock()

	appendToFile(logFileName, logEntry)
}

// logResponse logs the outgoing response headers and body in plain text
func logResponse(logFileName string, resp *http.Response) {
	logEntry := fmt.Sprintf("Outgoing Response: %v\n", resp.Status)

	// Log headers
	for name, values := range resp.Header {
		for _, value := range values {
			logEntry += fmt.Sprintf("Header: %v: %v\n", name, value)
		}
	}

	// Log body
	if resp.Body != nil {
		bodyBytes, _ := io.ReadAll(resp.Body)
		resp.Body = io.NopCloser(bytes.NewBuffer(bodyBytes)) // Reassign body for reuse
		logEntry += fmt.Sprintf("Body: %s\n", string(bodyBytes))
	}

	// Write the log entry to file
	logEntry += "\n"
	fileMutex.Lock()
	defer fileMutex.Unlock()

	appendToFile(logFileName, logEntry)
}
