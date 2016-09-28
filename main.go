package main

import (
    "fmt"
    "log"
    "net/http"
    "os"
    "io/ioutil"
//     "encoding/json"
)

type listData struct {
  ReposId   string    `json:"repos"`
  InboxId   string    `json:"inbox"`
  InWorksId string    `json:"works"`
  BlockedId string    `json:"block"`
  ReviewId  string    `json:"review"`
  MergedId  string    `json:"merged"`
  DeployId  string    `json:"deploy"`
  TestId    string    `json:"tested"`
  AcceptId  string    `json:"accept"`
}

func main() {
  /* Check if we are run to [re]-initialise the board */
  if (len(os.Args) >= 4) {
    // TODO make a proper class here
    key, token, boardid := os.Args[1], os.Args[2], os.Args[3]
    trello := NewTrello(key, token, boardid)
    
    trello.AddLabel("ErintLabs/io", "black")
    
//     /* Archive all open lists */
//     for _, v := range trello.ListIds() {
//       trello.CloseList(v)
//     }
// 
//     /* Ugly but effective, creating new lists */
//     listdata := listData{
//       trello.AddList("Repositories"),
//       trello.AddList("Inbox"),
//       trello.AddList("In Works"),
//       trello.AddList("Blocked"),
//       trello.AddList("Awaiting Review"),
//       trello.AddList("Merged to Mainline"),
//       trello.AddList("Deployed on Test"),
//       trello.AddList("Tested"),
//       trello.AddList("Accepted"),
//     }
//     
//     /* Happily print the JSON */
//     data, _ := json.Marshal(listdata)
//     fmt.Println("Set $LISTS to the following value:")
//     fmt.Println(string(data[:]))
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
