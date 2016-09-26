package main

import (
    "fmt"
    "log"
    "net/http"
    "os"
    "io/ioutil"
)

func main() {
  /* Check if we are run to [re]-initialise the board */
  if (len(os.Args) >= 4) {
    // TODO make a proper class here
    key, token, boardid := os.Args[1], os.Args[2], os.Args[3]
    baseurl := "https://api.trello.com/1"

    resp, err := http.Get(baseurl + "/boards/" + boardid + "?key=" + key + "&token=" + token)
    // TODO if error
    defer resp.Body.Close()
    body, err := ioutil.ReadAll(resp.Body)
    err = err

    fmt.Println(string(body[:]))

    /* Archive all lists */

    /* Check and activate GitHub powerup */

    /* Create the new lists */

    /* Happily print the JSON */
  } else {
    port := os.Getenv("PORT")

  	if port == "" {
  		log.Fatal("$PORT must be set")
  	}

    http.HandleFunc("/issues", Issues);
    log.Fatal(http.ListenAndServe(":"+port, nil))
  }
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
