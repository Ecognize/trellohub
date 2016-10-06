/* Operations with Trello lists */
package trello

import (
  . "../genapi"
  "net/url"
)

/* Adds a list to the board with a given name and returns the list id */
func (this *Trello) AddList(listname string) string {
  data := Object{}
  GenPOSTForm(this, "/lists/", &data, url.Values{
    "name": { listname },
    "idBoard": { this.BoardId },
    "pos": { "bottom" } })

  return data.Id
}

/* Lists all the open lists on the board */
func (this *Trello) ListIds() []string {
  var data []Object
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

