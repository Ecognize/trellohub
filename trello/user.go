/* Operations with Trello users */
package trello

import (
  . "github.com/ErintLabs/trellohub/genapi"
  "log"
  "net/url"
)

type tUser struct {
  Name  string    `json:"username"`
  Id    string    `json:"id"`
}

/* Check if a user is assigned to the card */
func (card *Card) loadMembers() {
  var users []tUser
  GenGET(card.trello, "/cards/" + card.Id + "/members", &users)

  card.Members = NewSet()

  /* TODO cache this one too */
  for _, v := range users {
    card.Members[v.Id] = true
  }
}

/* Resolve user names to ids */
func (trello *Trello) makeUserCache() {
  var members []tUser
  GenGET(trello, "/boards/" + trello.BoardId + "/members/", &members)

  for _, v := range members {
    trello.userIdbyName[v.Name] = v.Id
  }

  /* Generating a reverse one too */
  trello.userNamebyId = DicRev(trello.userIdbyName)
}

/* Wrapper around the dictionary not to expose */
func (trello *Trello) UserById(userid string) string {
  return trello.userNamebyId[userid]
}

func (trello *Trello) UserByName(username string) string {
  return trello.userIdbyName[username]
}

/* Assign/Unassign a user to the card */
func (card *Card) AddUser(user string) {
  log.Printf("Adding user %s to card %s.", user, card.Id)
  GenPOSTForm(card.trello, "/cards/" + card.Id + "/idMembers", nil, url.Values{ "value": { card.trello.userIdbyName[user] } })
}

func (card *Card) DelUser(user string) {
  log.Printf("Removing user %s from card %s.", user, card.Id)
  GenDEL(card.trello, "/cards/" + card.Id + "/idMembers/" + card.trello.userIdbyName[user])
}
