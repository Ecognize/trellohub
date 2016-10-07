/* Operations with Trello users */
package trello

import (
  . "../genapi"
  "log"
  "net/url"
)

type tUser struct {
  Name  string    `json:"username"`
  Id    string    `json:"id"`
}

/* Check if a user is assigned to the card */
func (this *Trello) UserAssigned(user string, cardid string) bool {
  var users []tUser
  GenGET(this, "/cards/" + cardid + "/members", &users)

  /* TODO cache this one too */
  for _, v := range users {
    if v.Name == user {
      return true
    }
  }

  return false
}

/* Resolve user names to ids */
func (this *Trello) makeUserCache() {
  var members []tUser
  GenGET(this, "/boards/" + this.BoardId + "/members/", &members)

  for _, v := range members {
    this.userCache[v.Name] = v.Id
  }

  /* Generating a reverse one too */
  this.userIdCache = DicRev(this.userCache)
}

/* Wrapper around the dictionary not to expose */
func (this *Trello) UserById(userid string) string {
  return this.userIdCache[userid]
}

/* Assign/Unassign a user to the card */
func (this *Trello) AddUser(user string, cardid string) {
  log.Printf("Adding user %s to card %s.", user, cardid)
  GenPOSTForm(this, "/cards/" + cardid + "/idMembers", nil, url.Values{ "value": { this.userCache[user] } })
}

func (this *Trello) DelUser(user string, cardid string) {
  log.Printf("Removing user %s from card %s.", user, cardid)
  GenDEL(this, "/cards/" + cardid + "/idMembers/" + this.userCache[user])
}
