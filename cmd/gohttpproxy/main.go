package main

import (
	"bufio"
	"fmt"
	"log"
	"net/http"
	"net/http/httputil"
	"net/url"
	"os"
	"strings"
	"sync"
)

// Mutex for file writes to avoid concurrency issues
var fileMutex sync.Mutex

// JSON output schema
type LogEntry struct {
	Net    NetInfo  `json:"net"`
	Header []string `json:"header"`
	Body   string   `json:"body"`
}

type LogWrapper struct {
	Logs []LogEntry `json:"logs"`
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
