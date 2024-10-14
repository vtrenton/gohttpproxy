package main

import (
	"bufio"
	"fmt"
	"os"
	"strings"
)

func main() {
	const localAddr = "127.0.0.1"

	if len(os.Args) != 3 {
		reader := bufio.NewReader(os.Stdin)
		fmt.Print("which port would you like this to run on? ")
		lport, _ := reader.ReadString('\n')
		lport = strings.TrimSpace(lport)

		fmt.Print("What remote address should this connect to? ")
		rhost, _ := reader.ReadString('\n')
	} else {
		lport := os.Args[1]
		rsock := os.Args[2]
		sockInd := strings.Index(rsock, ":")
		rhost := rsock[:sockInd]
		rport := rsock[sockInd+1:]
	}

	lportval := validatePort(localAddr, lport)

	// start http server
	// capture traffic and write to file
	// forward traffic to endpoint
}

func validatePort(address, port string) bool {
	if len(port) < 1 {
		log.Ffatalf("Port cannot be empty")
	}
}
