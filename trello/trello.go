package trello

import (
  . "../genapi"
  "net/url"
  "log"
)

// TODO: handle error responces from Trello

/* TODO comments */
type ListRef struct {
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

type CheckItem struct {
  Checked bool
  Text    string
}

type Payload struct {
  Action      struct {
    Type      string        `json:"type"`
    Data      struct {
      Member  string        `json:"idMember"`
      List    Object        `json:"list"`
      Card    Object        `json:"card"`
      ListB   Object        `json:"listBefore"`
      ListA   Object        `json:"listAfter"`
      Attach  struct {
        URL   string        `json:"url"`
      }                     `json:"attachment"`
    }                       `json:"data"`
  }                         `json:"action"`
}

/* Cascading types */
type Id struct {
  Id      string    `json:"id"`
}

type Object struct {
  Id
  Name    string    `json:"name"`
}

/* TODO make some fields private */
type Trello struct {
  Token string
  Key string
  BoardId string
  Lists ListRef

  /* RenameThese to make sense */
  labelCache map[string]string
  userCache map[string]string
  userIdCache map[string]string

  cardById      map[string]*Card
  cardByIssue   map[string]*Card
}

func New(key string, token string, boardid string) *Trello {
  t := new(Trello)
  t.Token = token
  t.Key = key

  t.BoardId = t.getFullBoardId(boardid)

  return t
}

func (this *Trello) Startup() {
  this.labelCache = make(map[string]string)
  this.makeLabelCache()

  /* Note: we assume users don't change anyway so we only do this at startup */
  this.userCache = make(map[string]string)
  this.makeUserCache()

  // TODO make cardCache
}

func (this *Trello) AuthQuery() string {
  return "key=" + this.Key + "&token=" + this.Token
}

func (this *Trello) BaseURL() string {
  return "https://api.trello.com/1"
}

func (this *Trello) getFullBoardId(boardid string) string {
  data := Object{}
  GenGET(this, "/boards/" + boardid, &data)
  return data.Id
}

type webhookInfo struct {
  Id    string    `json:"id"`
  Model string    `json:"idModel"`
  URL   string    `json:"callbackURL"`
}

/* Checks that a webhook is installed over the board, in case it isn't creates one */
func (this *Trello) EnsureHook(callbackURL string) {
  /* Check if we have a hook already */
  var data []webhookInfo
  GenGET(this, "/token/" + this.Token + "/webhooks/", &data)
  found := false

  for _, v := range data {
    /* Check if we have a hook for our own URL at same model */
    if v.Model == this.BoardId {
      if v.URL == callbackURL {
        log.Print("Hook found, nothing to do here.")
        found = true
        break
      }
    }
  }

  /* If not, install one */
  if !found {
    /* TODO: save hook reference and uninstall maybe? */
    GenPOSTForm(this, "/webhooks/", nil, url.Values{
      "name": { "trellohub for " + this.BoardId },
      "idModel": { this.BoardId },
      "callbackURL": { callbackURL } })

    log.Print("Webhook installed.")
  } else {
    log.Print("Reusing existing webhook.")
  }
}
