/* Operations with Trello cards */
package trello

import (
  . "../genapi"
  "net/url"
  "log"
  "strconv"
  "regexp"
)

/* Adds a card to the list with a given name and returns the card id */
func (this *Trello) AddCard(listid string, name string, desc string) string {
  data := Object{}
  GenPOSTForm(this, "/cards/", &data, url.Values{
    "name": { name },
    "idList": { listid },
    "desc": { desc },
    "pos": { "top" } })

  return data.Id
}

/* Attache a named URL to the card */
func (this *Trello) AttachURL(cardid string, addr string) {
  GenPOSTForm(this, "/cards/" + cardid + "/attachments", nil, url.Values{ "url": { addr } })
}

/* Returns the list currently containing the card */
func (this *Trello) CardList(cardid string) string {
  data := Object{}
  GenGET(this, "/cards/" + cardid + "/list/", &data)
  return data.Id
}

/* Move a card to the different list */
func (this *Trello) MoveCard(cardid string, listid string) {
  log.Printf("Moving card %s to list %s.", cardid, listid)
  GenPUT(this, "/cards/" + cardid + "/idList?value=" + listid)
}

/* Returns the name(technically URL) of the first attachment to the card */
func (this *Trello) FirstLink(cardid string) string {
  var data []Object
  GenGET(this, "/cards/" + cardid + "/attachments/", &data)

  if len(data) > 0 {
    return data[0].Name
  }
  return ""
}

/* Find card by Issue. Assuming only one such card exists. */
// TODO: seriously, implement caching here!
func (this *Trello) FindCard(issue IssueSpec) string {
  var data []Object
  GenGET(this, "/boards/" + this.BoardId + "/cards/", &data)

  /* Picking the one we need */
  ref := issue.rid + "/issues/" + strconv.Itoa(issue.iid)
  for _, v := range data {
    if ok, _ := regexp.MatchString(ref, this.FirstLink(v.Id)); ok {
      return v.Id
    }
  }

  /* Sorry, no */
  return ""
}

