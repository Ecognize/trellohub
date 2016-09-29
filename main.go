package main

import (
    "fmt"
    "log"
    "net/http"
    "os"
    "io/ioutil"
    "encoding/json"
    "regexp"
)

/* TODO: move to GitHub */
type IssuePayload struct {
  Action  string    `json:"action"`
  Issue struct {
    URL   string    `json:"html_url"`
    Title string    `json:"title"`
    Body  string    `json:"body"`
  }                 `json:"issue"`
}

/* TODO: move to Trello */
type TrelloPayload struct {
  Action      struct {
    Type      string        `json:"type"`
    Data      struct {
      List    TrelloObject  `json:"list"`
      Card    TrelloObject  `json:"card"`
      Attach  struct {
        URL   string        `json:"url"`
      }                     `json:"attachment"`
    }                       `json:"data"`
  }                         `json:"action"`
}

/* Globals are bad */
var trello *Trello
var github *GitHub;
const REGEX_GH_REPO string = "^(https?://)?github.com/([^/]*)/([^/]*)"

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
    base_url := os.Getenv("URL")
    trello = NewTrello(trello_key, trello_token, boardid)

    json.Unmarshal([]byte(os.Getenv("LISTS")), &trello.Lists)

    /* TODO extend for other params */
  	if port == "" {
  		log.Fatal("$PORT must be set")
  	}

    /* Registering handlers */
    http.HandleFunc("/trello", TrelloFunc)
    http.HandleFunc("/trello/", TrelloFunc)

    http.HandleFunc("/issues", IssuesFunc)
    http.HandleFunc("/issues/", IssuesFunc)

    http.HandleFunc("/pull", PullFunc)
    http.HandleFunc("/pull/", PullFunc)

    /* Ensuring Trello hook */
    /* TODO: study if this doesn't cause races */
    go trello.EnsureHook(base_url + "/trello")

    /* Starting the server up */
    log.Fatal(http.ListenAndServe(":"+port, nil))
  }
}

type handleSubroutine func (body []byte) (int, string)

func GeneralisedProcess(w http.ResponseWriter, r *http.Request, f handleSubroutine) {
  // TODO io.LimitReader
  // TODO check if its or POST
  body, err := ioutil.ReadAll(r.Body)
  if err != nil {
      log.Fatal(err)
  }

  /* Invoking the actual function */
  //log.Print(string(body[:]))
  var code int
  var text string

  if r.Method != "HEAD" {
    code, text = f(body)
  } else { /* or not, if it's a HEAD */
    code, text = http.StatusOK, "Pleased to meet you."
  }

  /* Replying to the caller */
  w.WriteHeader(code)
  fmt.Fprintln(w, text)

  /* Finalise session */
  if err := r.Body.Close(); err != nil {
      log.Fatal(err)
  }
}

func TrelloFunc(w http.ResponseWriter, r *http.Request) {
  GeneralisedProcess(w, r, func (body []byte) (int, string) {
    event := TrelloPayload{}
    json.Unmarshal(body, &event)

    /* TODO: switch */
    if event.Action.Type == "addAttachmentToCard" {
      // TODO: also install GitHub webhooks when possible
      /* Check if the list is correct */
      if trello.CardList(event.Action.Data.Card.Id) == trello.Lists.ReposId {
        /* Check if this is a GitHub URL after all */
        re := regexp.MustCompile(REGEX_GH_REPO)
        if res := re.FindStringSubmatch(event.Action.Data.Attach.URL); res != nil {
          repoid := res[2] + "/" + res[3]
          log.Printf("Registering new repository: %s.", repoid)

          /* Add a label, but make sure no duplicates happen */
          if trello.GetLabel(repoid) == "" {
            trello.SetLabel(event.Action.Data.Card.Id, trello.AddLabel(repoid))
          } else {
            log.Print("Label already there, not proceeding.")
          }
        }
      }

      return http.StatusOK, "Attachment processed."
    }

    //log.Print(string(body[:]))
    return http.StatusOK, "Erm, hello"
  })
}

func IssuesFunc(w http.ResponseWriter, r *http.Request) {
  GeneralisedProcess(w, r, func (body []byte) (int, string) {
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
        cardid := trello.AddCard(trello.Lists.InboxId, issue.Issue.Title, issue.Issue.Body)
        trello.AttachURL(cardid, issue.Issue.URL)
        trello.SetLabel(cardid, labelid)

        /* Happily report */
        log.Printf("Creating card %s for issue %s\n", cardid, issue.Issue.URL)
        return http.StatusOK, "Got your back, captain."
      } else {
        return http.StatusNotFound, "You sure we serve this repo? I don't think so."
      }
    }
    return http.StatusOK, "I can't really process this, but fine."
  })
}

func PullFunc(w http.ResponseWriter, r *http.Request) {
  GeneralisedProcess(w, r, func (body []byte) (int, string) {
    log.Print(string(body[:]))

    return http.StatusOK, "I can't really process this, but fine."
  })
}
