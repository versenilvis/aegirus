// NOTE: FOR DEVELOPMENT ONLY
// This is a mock server to test your application
// You can run this server and test your application
// with curl or Postman
package main

import (
	"fmt"
	"log"
	"net/http"
	"sync"
)

func main() {
	var wg sync.WaitGroup
	wg.Add(2)

	// Server A (3001)
	go func() {
		defer wg.Done()
		mux := http.NewServeMux()
		mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
			fmt.Println("--> Server A (3001) received a request")
			w.Write([]byte("Hello from SERVER A (Port 3001)"))
		})
		log.Println("Mock Server A running on :3001")
		http.ListenAndServe(":3001", mux)
	}()

	// Server B (3002)
	go func() {
		defer wg.Done()
		mux := http.NewServeMux()
		mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
			fmt.Println("--> Server B (3002) received a request")
			w.Write([]byte("Hello from SERVER B (Port 3002)"))
		})
		log.Println("Mock Server B running on :3002")
		http.ListenAndServe(":3002", mux)
	}()

	wg.Wait()
}
