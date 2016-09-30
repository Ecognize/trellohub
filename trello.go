package main

import (
  "net/url"
  "log"
  "regexp"
  "strconv"
)

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

/* TODO make private */
type Trello struct {
  Token string
  Key string
  BoardId string
  Lists ListRef
  labelCache map[string]string
}

func NewTrello(key string, token string, boardid string) *Trello {
  t := new(Trello)
  t.Token = token
  t.Key = key

  t.BoardId = t.getFullBoardId(boardid)
  t.labelCache = make(map[string]string)

  t.makeLabelCache()

  return t
}

func (this *Trello) AuthQuery() string {
  return "key=" + this.Key + "&token=" + this.Token
}

func (this *Trello) BaseURL() string {
  return "https://api.trello.com/1"
}

type TrelloObject struct {
  Id      string    `json:"id"`
  Name    string    `json:"name"`
}

type webhookInfo struct {
    Id    string    `json:"id"`
    Model string    `json:"idModel"`
    URL   string    `json:"callbackURL"`
}

func (this *Trello) getFullBoardId(boardid string) string {
  data := TrelloObject{}
  GenGET(this, "/boards/" + boardid, &data)
  return data.Id
}

/* Adds a list to the board with a given name and returns the list id */
func (this *Trello) AddList(listname string) string {
  data := TrelloObject{}
  GenPOSTForm(this, "/lists/", &data, url.Values{
    "name": { listname },
    "idBoard": { this.BoardId },
    "pos": { "bottom" } })

  return data.Id
}

/* Adds a card to the list with a given name and returns the card id */
func (this *Trello) AddCard(listid string, name string, desc string) string {
  data := TrelloObject{}
  GenPOSTForm(this, "/cards/", &data, url.Values{
    "name": { name },
    "idList": { listid },
    "desc": { desc },
    "pos": { "top" } })

  return data.Id
}

/* Lists all the open lists on the board */
func (this *Trello) ListIds() []string {
  var data []TrelloObject
  GenGET(this, "/boards/" + this.BoardId + "/lists/?filter=open", &data)
  res := make([]string, len(data))
  for i, v := range data {
    res[i] = v.Id
  }
  return res
}

/* Archive a list */
func (this *Trello) CloseList(listid string) {
  GenPUT(this, "/lists/" + listid + "/closed?value=true")
}

/* Attache a named URL to the card */
func (this *Trello) AttachURL(cardid string, addr string) {
  GenPOSTForm(this, "/cards/" + cardid + "/attachments", nil, url.Values{ "url": { addr } })
}

/* Returns the list currently containing the card */
func (this *Trello) CardList(cardid string) string {
  data := TrelloObject{}
  GenGET(this, "/cards/" + cardid + "/list/", &data)
  return data.Id
}

/* Move a card to the different list */
func (this *Trello) MoveCard(cardid string, listid string) {
  log.Printf("Moving card %s to list %s.", cardid, listid)
  GenPUT(this, "/cards/" + cardid + "/idList?value=" + listid)
}

/* Add a label to board */
func (this *Trello) AddLabel(name string) string {
  /* Pick up a color first */
  colors := [...]string { "green", "yellow", "orange", "red", "purple", "blue", "sky", "lime", "pink", "black" }

  var labels []TrelloObject
  GenGET(this, "/boards/" + this.BoardId + "/labels/", &labels)

  /* TODO: avoid duplicates too */

  /* Create a label with appropriate color */
  col := colors[ (len(labels)-6) % len(colors) ]
  log.Printf("Creating a new %s label name %s in Trello.", col, name)
  data := TrelloObject{}
  GenPOSTForm(this, "/labels/", &data, url.Values{
    "name": { name },
    "idBoard": { this.BoardId },
    "color": { col } })

  this.labelCache[name] = data.Id 

  return data.Id
}

/* Attach a label to the card */
func (this *Trello) SetLabel(cardid string, labelid string) {
    GenPOSTForm(this, "/cards/" + cardid + "/idLabels", nil, url.Values{ "value": { labelid } })
}

/* Build a repo to label correspondence cache */
func (this *Trello) makeLabelCache() bool {
  var labels []TrelloObject
  GenGET(this, "/boards/" + this.BoardId + "/labels/", &labels)

  for _, v := range labels {
    this.labelCache[v.Name] = v.Id
  }

  return true // needed for dirty magic
}

/* Get the label id or empty string if not found */
func (this *Trello) GetLabel(repoid string) string {
  /* Look in cache, if not there retry */
  for updated := false; !updated; updated = this.makeLabelCache() {
    if id, ok := this.labelCache[repoid]; ok {
      return id
    }
  }

  /* If we are still there, something's wrong */
  return ""
}

/* Looks up a label to corresponding repository, returns an empty string if not found */
func (this *Trello) FindLabel(addr string) string {
  /* Break the incoming string down to just Owner/repo */
  var key string
  re := regexp.MustCompile(REGEX_GH_REPO)
  if res := re.FindStringSubmatch(addr); res != nil {
    key = res[2] + "/" + res[3]
  } else {
    log.Fatal("Incoming URL fails GitHubness, what's going on?")
    return ""
  }

  return this.GetLabel(key)
}

/* Checks that a webhook is installed over the board, in case it isn't creates one */
func (this *Trello) EnsureHook(callbackURL string) {
  /* Check if we have a hook already */
  var data []webhookInfo
  GenGET(this, "/token/" + this.Token + "/webhooks/", &data)
  found := false

  for _, v := range data {
    /* Remove a hook if it points to some other URL, but same Model */
    if v.Model == this.BoardId {
      if v.URL != callbackURL {
        log.Printf("Found a hook on a different URL: %s. Removing.", v.URL)
        GenDEL(this, "/webhooks/" + v.Id)
      } else {
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

/* Returns the name(technically URL) of the first attachment to the card */
func (this *Trello) FirstLink(cardid string) string {
  var data []TrelloObject
  GenGET(this, "/cards/" + cardid + "/attachments/", &data)

  if len(data) > 0 {
    return data[0].Name
  }
  return ""
}

/* Find card by Issue. Assuming only one such card exists. */
// TODO: seriously, implement caching here!
func (this *Trello) FindCard(issue IssueSpec) string {
  var data []TrelloObject
  GenGET(this, "/boards/" + this.BoardId + "/cards/", &data)

  /* Picking the one we need */
  ref := issue.rid + "/issues/" + strconv.Itoa(issue.iid)
  for _, v := range data {
    if ok, _ := regexp.MatchString(ref, trello.FirstLink(v.Id)); ok {
      return v.Id
    }
  }

  /* Sorry, no */
  return ""
}
