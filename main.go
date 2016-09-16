package main

import (
    "fmt"
    "log"
    "net/http"
    "os"
    "io/ioutil"
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
  // TODO io.LimitReader
    body, err := ioutil.ReadAll(r.Body)
    if err != nil {
        log.Fatal(err)
    }

    fmt.Fprintln(w, "")
    if err := r.Body.Close(); err != nil {
        log.Fatal(err)
    }

    log.Print(string(body[:]))
}
