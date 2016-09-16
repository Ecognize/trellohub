package main

import (
    "fmt"
    "log"
    "net/http"
    "os"
)

func main() {
  port := os.Getenv("PORT")

	if port == "" {
		log.Fatal("$PORT must be set")
	}

  http.HandleFunc("/issues", Issues);
  log.Fatal(http.ListenAndServe(":"+port, nil))
}

func Issues(w http.ResponseWriter, r *http.Request) {
    fmt.Fprintln(w, "Welcome! "+r.UserAgent() )
}
