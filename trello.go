package main

import (
  "fmt"
)

/* TODO make private */
type Trello struct {
  Token string
  Key string
}

func NewTrello(key string, token string) *Trello {
  t := new(Trello)
  t.Token = token
  t.Key = key

  return t
}

func (this *Trello) AuthQuery() string {
  return "?key=" + this.Key + "&token=" + this.Token
}

func (this *Trello) BaseURL() string {
  return "https://api.trello.com/1"
}

type boardData struct {
  Id  string    `json:id`
  Name string   `json:name`
}

func (this *Trello) Get(rq string) {
  data := boardData{}
  GenGET(this, rq, &data)
  fmt.Println(data.Name)
}
