/* Operations with Trello lists */
package trello

import (
  . "../genapi"
  "net/url"
)

type List struct {
  Object
  trello *Trello
}

/* Adds a list to the board with a given name and returns the list id */
func (trello *Trello) AddList(listname string) string {
  data := Object{}
  GenPOSTForm(trello, "/lists/", &data, url.Values{
    "name": { listname },
    "idBoard": { trello.BoardId },
    "pos": { "bottom" } })

  return data.Id
}

/* Lists all the open lists on the board */
func (trello *Trello) GetLists() []List {
  var data []List
  GenGET(trello, "/boards/" + trello.BoardId + "/lists/?filter=open", &data)

  for i,_ := range data {
    data[i].trello = trello
  }

  return data
}

/* Archive a list */
func (list *List) Close() {
  GenPUT(list.trello, "/lists/" + list.Id + "/closed?value=true")
}
