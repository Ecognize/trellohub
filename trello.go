package main

import (
  "net/url"
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
}

func NewTrello(key string, token string, boardid string) *Trello {
  t := new(Trello)
  t.Token = token
  t.Key = key

  t.BoardId = t.getFullBoardId(boardid)

  return t
}

func (this *Trello) AuthQuery() string {
  return "key=" + this.Key + "&token=" + this.Token
}

func (this *Trello) BaseURL() string {
  return "https://api.trello.com/1"
}

type namedEntity struct {
  Id      string    `json:"id"`
  Name    string    `json:"name"`
}

func (this *Trello) getFullBoardId(boardid string) string {
  data := namedEntity{}
  GenGET(this, "/boards/" + boardid, &data)
  return data.Id
}

/* Adds a list to the board with a given name and returns the list id */
func (this *Trello) AddList(listname string) string {
  data := namedEntity{}
  GenPOSTForm(this, "/lists/", &data, url.Values{
    "name": { listname },
    "idBoard": { this.BoardId },
    "pos": { "bottom" } })

  return data.Id
}

/* Adds a card to the list with a given name and returns the card id */
func (this *Trello) AddCard(listid string, cardname string) string {
  data := namedEntity{}
  GenPOSTForm(this, "/cards/", &data, url.Values{
    "name": { cardname },
    "idList": { listid },
    "pos": { "top" } })

  return data.Id
}

/* Lists all the open lists on the board */
func (this *Trello) ListIds() []string {
  var data []namedEntity
  GenGET(this, "/boards/" + this.BoardId + "/lists/?filter=open", &data)
  res := make([]string, len(data))
  for i, v := range data {
    res[i] = v.Id
  }
  return res
}

/* Archives a list */
func (this *Trello) CloseList(listid string) {
  GenPUT(this, "/lists/" + listid + "/closed?value=true")
}

/* Attaches a named URL to the card */
func (this *Trello) AttachURL(cardid string, addr string) {
  GenPOSTForm(this, "/cards/" + cardid + "/attachments", nil, url.Values{ "url": { addr } })
}

/* Move a card to the different list */
func (this *Trello) MoveCard(cardid string, listid string) {
  GenPUT(this, "/cards/" + cardid + "/idList?value=" + listid)
}

/* Add a label to board */
func (this *Trello) AddLabel(name string) string {
  /* Pick up a color first */
  colors := [...]string { "green", "yellow", "orange", "red", "purple", "blue", "sky", "lime", "pink", "black" }
  
  var labels []namedEntity
  GenGET(this, "/boards/" + this.BoardId + "/labels/", &labels)
  
  /* TODO: avoid duplicates too */
  
  /* Create a label with appropriate color */
  data := namedEntity{}
  GenPOSTForm(this, "/labels/", &data, url.Values{
    "name": { name },
    "idBoard": { this.BoardId },
    "color": { colors[ (len(labels)-6) % len(colors) ] } })

  return data.Id
}

/* Attach a label to the card */
func (this *Trello) SetLabel(cardid string, labelid string) {
    GenPOSTForm(this, "/cards/" + cardid + "/idLabels", nil, url.Values{ "value": { labelid } })
}
