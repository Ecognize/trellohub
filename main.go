package main

import (
    "fmt"
    "log"
    "net/http"
    "os"
    "io/ioutil"
    "encoding/json"
)

/* TODO: move to GitHub */
type IssuePayload struct {
  Action  string    `json:"action"`
  Issue struct {
    URL   string    `json:"html_url"`
    Title string    `json:"title"`
  }                 `json:"issue"`
}

/* Globals are bad */
var trello *Trello

func main() {
  /* Check if we are run to [re]-initialise the board */
  if (len(os.Args) >= 4) {
    key, token, boardid := os.Args[1], os.Args[2], os.Args[3]
    trello = NewTrello(key, token, boardid)

    /* Archive all open lists */
    for _, v := range trello.ListIds() {
      trello.CloseList(v)
    }

    /* Ugly but effective, creating new lists */
    trello.Lists = ListRef{
      trello.AddList("Repositories"),
      trello.AddList("Inbox"),
      trello.AddList("In Works"),
      trello.AddList("Blocked"),
      trello.AddList("Awaiting Review"),
      trello.AddList("Merged to Mainline"),
      trello.AddList("Deployed on Test"),
      trello.AddList("Tested"),
      trello.AddList("Accepted"),
    }

    /* Happily print the JSON */
    data, _ := json.Marshal(trello.Lists)
    fmt.Println("Set $LISTS to the following value:")
    fmt.Println(string(data[:]))
  } else {
    /* General config */
    port := os.Getenv("PORT")

    /* Trello config */
    trello_key, trello_token := os.Getenv("TRELLO_KEY"), os.Getenv("TRELLO_TOKEN")
    boardid := os.Getenv("BOARD")
    trello = NewTrello(trello_key, trello_token, boardid)

    json.Unmarshal([]byte(os.Getenv("LISTS")), &trello.Lists)

    /* TODO extend for other params */
  	if port == "" {
  		log.Fatal("$PORT must be set")
  	}

    http.HandleFunc("/issues", Issues);
    log.Fatal(http.ListenAndServe(":"+port, nil))
  }
}

func Issues(w http.ResponseWriter, r *http.Request) {
  // TODO io.LimitReader
  // TODO proper code eh?
  // TODO template method anyway
  // TODO check if its HEAD or POST
    body, err := ioutil.ReadAll(r.Body)
    if err != nil {
        log.Fatal(err)
    }

    /* TODO check json errors */
    /* TODO check it was github who sent it anyway */
    /* TODO check whether we serve this repo */
    var issue IssuePayload
    json.Unmarshal(body, &issue)

    /* Guess we have a new issue */
    if issue.Action == "opened" {
      /* Look up the corresponding label */
      if labelid := trello.FindLabel(issue.Issue.URL); len(labelid) > 0 {
        /* Insert the card, attach the issue and label */
        cardid := trello.AddCard(trello.Lists.InboxId, issue.Issue.Title)
        trello.AttachURL(cardid, issue.Issue.URL)
        trello.SetLabel(cardid, labelid)

        /* Happily report */
        log.Printf("Creating card %s for issue %s\n", cardid, issue.Issue.URL)
        w.WriteHeader(http.StatusOK)
        fmt.Fprintln(w, "Got your back, captain.")
      } else {
        w.WriteHeader(http.StatusNotFound)
        fmt.Fprintln(w, "You sure we serve this repo? I don't think so.")
      }
    }

    //log.Print(string(body[:]))


    /* Finalise session */
    if err := r.Body.Close(); err != nil {
        log.Fatal(err)
    }
}
