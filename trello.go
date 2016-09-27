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
  return "?key=" + this.Key + "&token=" + this.Token
}

func (this *Trello) BaseURL() string {
  return "https://api.trello.com/1"
}

type boardData struct {
  Id      string    `json:id`
  Name    string    `json:name`
}

func (this *Trello) getFullBoardId(boardid string) string {
  data := boardData{}
  GenGET(this, "/boards/" + boardid, &data)
  return data.Id
}

/* Adds a list to the board with a given name and returns the list id */
func (this *Trello) AddList(listname string) {
  data := boardData{}
  GenPOSTForm(this, "/lists/", data, url.Values{
    "name": { listname },
    "idBoard": { this.BoardId },
    "pos": { "top" } })
}
