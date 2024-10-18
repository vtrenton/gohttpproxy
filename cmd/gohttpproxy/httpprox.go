package main

import (
	"bufio"
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"io/ioutil"
	"log"
	"net"
	"net/http"
	"net/http/httputil"
	"net/url"
	"os"
	"strings"
	"sync"
	"time"
)

// Mutex for file writes to avoid concurrency issues
var fileMutex sync.Mutex

// JSON output schema
type LogEntry struct {
	Net    NetInfo  `json:"net"`
	Header []string `json:"header"`
	Body   string   `json:"body"`
}

type NetInfo struct {
	Source string `json:"source"`
	Dst    string `json:"dst"`
}

func main() {
	const lhost = "127.0.0.1"
	var lport, rhost, rport string
	var outputAsJSON bool

	// Check if the first argument is --json
	args := os.Args[1:]
	if len(args) > 0 && args[0] == "--json" {
		outputAsJSON = true
		args = args[1:] // Shift the args so we can use the remaining as the port and destination
	}

	// Prompt user for input if local port and remote address/port are not passed as arguments
	if len(args) != 2 {
		reader := bufio.NewReader(os.Stdin)

		fmt.Print("Which port would you like this to run on? ")
		lport, _ = reader.ReadString('\n')
		lport = strings.TrimSpace(lport)

		fmt.Print("What remote address should this connect to? ")
		rhost, _ = reader.ReadString('\n')
		rhost = strings.TrimSpace(rhost)

		fmt.Print("What remote port should this connect to? ")
		rport, _ = reader.ReadString('\n')
		rport = strings.TrimSpace(rport)
	} else {
		lport = args[0]
		rsock := args[1]
		sockInd := strings.Index(rsock, ":")
		rhost = rsock[:sockInd]
		rport = rsock[sockInd+1:]
	}

	// Validate the ports (logic not provided, assumed valid)
	lconnectval := validatePort(lhost, lport, true)
	rconnectval := validatePort(rhost, rport, false)
	if !lconnectval || !rconnectval {
		log.Fatal("Invalid port or host configuration.")
	}

	// Define the remote URL to proxy to
	remoteURL := fmt.Sprintf("http://%s:%s", rhost, rport)
	proxyURL, err := url.Parse(remoteURL)
	if err != nil {
		log.Fatal("Error parsing remote URL:", err)
	}

	// Create a log file (JSON or normal)
	var logFileName string
	if outputAsJSON {
		logFileName = "output.json"
	} else {
		logFileName = "proxy.log"
	}

	// Create the HTTP proxy
	proxy := &httputil.ReverseProxy{
		Director: func(req *http.Request) {
			req.URL.Scheme = proxyURL.Scheme
			req.URL.Host = proxyURL.Host
			req.URL.Path = proxyURL.Path
		},
		ModifyResponse: func(resp *http.Response) error {
			if outputAsJSON {
				logResponseAsJSON(logFileName, resp) // Log response in JSON
			} else {
				logResponse(logFileName, resp) // Log the response headers and body
			}
			return nil
		},
		ErrorHandler: func(w http.ResponseWriter, r *http.Request, err error) {
			log.Printf("http: proxy error: %v\n", err)
		},
	}

	// Wrap the proxy handler with a logger for request logging
	http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		// Log a short line to stdout for each incoming request
		sourceAddr := r.RemoteAddr // Source address
		destAddr := r.Host         // Destination address and port
		fullURL := r.URL.String()  // Full URL with path and query params
		fmt.Printf("Source: %s -> Dest: %s, URL: %s\n", sourceAddr, destAddr, fullURL)

		// Log the incoming request
		if outputAsJSON {
			logRequestAsJSON(logFileName, r) // Log request in JSON
		} else {
			logRequest(logFileName, r) // Log request in plain text
		}

		// Serve the proxy request
		proxy.ServeHTTP(w, r)
	})

	// Start the HTTP server
	log.Printf("Starting proxy server on %s:%s forwarding to %s\n", lhost, lport, remoteURL)
	err = http.ListenAndServe(lhost+":"+lport, nil)
	if err != nil {
		log.Fatal("Error starting server:", err)
	}
}

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

// appendToJSON appends the log entry to the JSON array
func appendToJSON(logFileName string, entry LogEntry) {
	var logEntries []LogEntry

	// Read existing JSON array from file
	if fileInfo, err := os.Stat(logFileName); err == nil && fileInfo.Size() > 0 {
		data, err := ioutil.ReadFile(logFileName)
		if err != nil {
			log.Fatal("Error reading JSON file:", err)
		}
		if err := json.Unmarshal(data, &logEntries); err != nil {
			log.Fatal("Error unmarshalling JSON file:", err)
		}
	}

	// Append new log entry
	logEntries = append(logEntries, entry)

	// Write the updated array back to the file atomically
	tmpFileName := logFileName + ".tmp"
	tmpFile, err := os.Create(tmpFileName)
	if err != nil {
		log.Fatal("Error creating temp file:", err)
	}
	defer tmpFile.Close()

	encoder := json.NewEncoder(tmpFile)
	encoder.SetIndent("", "  ")
	if err := encoder.Encode(logEntries); err != nil {
		log.Fatal("Error writing JSON to file:", err)
	}

	// Atomically replace the original file
	if err := os.Rename(tmpFileName, logFileName); err != nil {
		log.Fatal("Error replacing the JSON file:", err)
	}
}

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
