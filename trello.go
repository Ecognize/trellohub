package main

import (
  "net/url"
)

/* TODO comments */

/* TODO make private */
type Trello struct {
  Token string
  Key string
  BoardId string
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

