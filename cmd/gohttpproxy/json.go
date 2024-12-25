package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"io/ioutil"
	"log"
	"net/http"
	"os"
)

// logRequestAsJSON logs the incoming request in JSON format
func logRequestAsJSON(logFileName string, r *http.Request) {
	headers := []string{}
	for name, values := range r.Header {
		for _, value := range values {
			// Sanitize and ensure proper formatting
			headerString := fmt.Sprintf("%v: %v", name, value)
			headers = append(headers, headerString)
		}
	}

	var body string
	if r.Body != nil {
		bodyBytes, _ := io.ReadAll(r.Body)
		r.Body = io.NopCloser(bytes.NewBuffer(bodyBytes))
		body = string(bodyBytes)
	}

	entry := LogEntry{
		Net: NetInfo{
			Source: r.RemoteAddr,
			Dst:    r.Host,
		},
		Header: headers,
		Body:   body,
	}

	// Append to JSON array with mutex
	fileMutex.Lock()
	defer fileMutex.Unlock()

	appendToJSON(logFileName, entry)
}

// logResponseAsJSON logs the outgoing response in JSON format
func logResponseAsJSON(logFileName string, resp *http.Response) {
	headers := []string{}
	for name, values := range resp.Header {
		for _, value := range values {
			// Sanitize and ensure proper formatting
			headerString := fmt.Sprintf("%v: %v", name, value)
			headers = append(headers, headerString)
		}
	}

	var body string
	if resp.Body != nil {
		bodyBytes, _ := io.ReadAll(resp.Body)
		resp.Body = io.NopCloser(bytes.NewBuffer(bodyBytes))
		body = string(bodyBytes)
	}

	entry := LogEntry{
		Net: NetInfo{
			Source: "proxy",
			Dst:    resp.Request.Host,
		},
		Header: headers,
		Body:   body,
	}

	// Append to JSON array with mutex
	fileMutex.Lock()
	defer fileMutex.Unlock()

	appendToJSON(logFileName, entry)
}

// appendToJSON appends the log entry to the JSON array, ensuring {} wrapping
func appendToJSON(logFileName string, entry LogEntry) {
	var logWrapper LogWrapper

	// Read existing JSON data from file
	if fileInfo, err := os.Stat(logFileName); err == nil && fileInfo.Size() > 0 {
		data, err := ioutil.ReadFile(logFileName)
		if err != nil {
			log.Fatal("Error reading JSON file:", err)
		}
		if err := json.Unmarshal(data, &logWrapper); err != nil {
			log.Fatal("Error unmarshalling JSON file:", err)
		}
	} else {
		logWrapper.Logs = []LogEntry{} // Initialize if file is empty or doesn't exist
	}

	// Append new log entry
	logWrapper.Logs = append(logWrapper.Logs, entry)

	// Write the updated JSON object back to the file atomically
	tmpFileName := logFileName + ".tmp"
	tmpFile, err := os.Create(tmpFileName)
	if err != nil {
		log.Fatal("Error creating temp file:", err)
	}
	defer tmpFile.Close()

	encoder := json.NewEncoder(tmpFile)
	encoder.SetIndent("", "  ")
	if err := encoder.Encode(logWrapper); err != nil {
		log.Fatal("Error writing JSON to file:", err)
	}

	// Atomically replace the original file
	if err := os.Rename(tmpFileName, logFileName); err != nil {
		log.Fatal("Error replacing the JSON file:", err)
	}
}
