package main

import (
	"bufio"
	"bytes"
	"fmt"
	"io"
	"log"
	"net/http"
	"net/http/httputil"
	"net/url"
	"os"
	"strings"
)

// validatePort is a placeholder for the port validation logic
func validatePort(host, port string) bool {
	// In practice, add validation logic here
	return true
}

func main() {
	const lhost = "127.0.0.1"
	var lport, rhost, rport string

	// Prompt user for input if not passed as arguments
	if len(os.Args) != 3 {
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
		lport = os.Args[1]
		rsock := os.Args[2]
		sockInd := strings.Index(rsock, ":")
		rhost = rsock[:sockInd]
		rport = rsock[sockInd+1:]
	}

	// Validate the ports (logic not provided, assumed valid)
	lconnectval := validatePort(lhost, lport)
	rconnectval := validatePort(rhost, rport)
	if !lconnectval || !rconnectval {
		log.Fatal("Invalid port or host configuration.")
	}

	// Define the remote URL to proxy to
	remoteURL := fmt.Sprintf("http://%s:%s", rhost, rport)
	proxyURL, err := url.Parse(remoteURL)
	if err != nil {
		log.Fatal("Error parsing remote URL:", err)
	}

	// Create a log file
	logFile, err := os.OpenFile("proxy.log", os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0666)
	if err != nil {
		log.Fatal("Failed to open log file:", err)
	}
	defer logFile.Close()

	// Create the HTTP proxy
	proxy := &httputil.ReverseProxy{
		Director: func(req *http.Request) {
			req.URL.Scheme = proxyURL.Scheme
			req.URL.Host = proxyURL.Host
			req.URL.Path = proxyURL.Path
		},
		ModifyResponse: func(resp *http.Response) error {
			logResponse(logFile, resp) // Log the response headers and body
			return nil
		},
	}

	// Wrap the proxy handler with a logger for request logging
	http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		sourceAddr := r.RemoteAddr
		destAddr := r.Host
		fullURL := r.URL.String()
		fmt.Printf("Src: %s -> Dst: %s, URL: %s\n", sourceAddr, destAddr, fullURL)

		// Log the incoming request headers and body
		logRequest(logFile, r)

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

// logRequest logs the incoming request headers and body
func logRequest(logFile *os.File, r *http.Request) {
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
	_, _ = logFile.WriteString(logEntry)
}

// logResponse logs the outgoing response headers and body
func logResponse(logFile *os.File, resp *http.Response) {
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
	_, _ = logFile.WriteString(logEntry)
}
