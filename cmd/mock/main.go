// NOTE: FOR DEVELOPMENT ONLY
// This is a mock server to test your application
// You can run this server and test your application
// with curl or Postman
package main

import (
	"flag"
	"fmt"
	"log"
	"net/http"
	"os"
)

func main() {
	port := flag.String("port", "3001", "Port to run server on")
	name := flag.String("name", "A", "Name of the server")
	flag.Parse()

	addr := ":" + *port

	mux := http.NewServeMux()
	mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		fmt.Printf("--> [%s] Server %s received a request\n", *port, *name)

		w.WriteHeader(http.StatusOK)
		w.Write([]byte(fmt.Sprintf("Hello from SERVER %s (Port %s)", *name, *port)))
	})

	log.Printf("Mock Server %s is running on http://127.0.0.1%s", *name, addr)
	log.Printf("   (Press Ctrl+C to stop this server)")

	err := http.ListenAndServe(addr, mux)
	if err != nil {
		log.Printf("Error starting server: %v\n", err)
		os.Exit(1)
	}
}
