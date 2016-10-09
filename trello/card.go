/* Operations with Trello cards */
package trello

import (
  . "../genapi"
  "../github"
  "net/url"
  "log"
  "strconv"
  "regexp"
)

type Card struct {
  // Object TODO cascading
  Id          string      `json:"id"`
  Name        string      `json:"name"`
  ListId      string      `json:"idList"`
  Desc        string      `json:"desc"`
  trello      *Trello
  issue       *github.Issue
  checklist   *Checklist
  Members     Set
}

/* Places the card in the cache */
func (card *Card) cache() {
  card.trello.cardById[card.Id] = card
}

/* Updates card data from the server */
func (card *Card) update() {
  GenGET(card.trello, "/cards/" + card.Id, card)
  // TODO if error

  /* We don't really care to hold attachments array, just check if there is something to link */
  var data []Object
  GenGET(card.trello, "/cards/" + card.Id + "/attachments/", &data)
  issuesFound := 0
  for _, v := range data {
    re := regexp.MustCompile(REGEX_GH_ISSUE)
    if res := re.FindStringSubmatch(v.Name); res != nil {
      issuesFound ++;
      if issuesFound > 1 {
        log.Printf("WARNING: Duplicate issue attachments found on card #%s.", card.Id)
      } else {
        issueno, _ := strconv.Atoi(res[2])
        card.LinkIssue(card.trello.github.GetIssue(res[1], issueno))
      }
    }
  }

  card.updateMembers()

  // REFACTOR: checklist
}

/* Adds a card to the list with a given name and returns the card id */
func (trello *Trello) AddCard(listid string, name string, desc string) *Card {
  data := Card{}
  GenPOSTForm(trello, "/cards/", &data, url.Values{
    "name": { name },
    "idList": { listid },
    "desc": { desc },
    "pos": { "top" } })

  data.cache()
  // TODO if error

  return &data
}

/* Retrieves the card from the server */
func (trello *Trello) GetCard(cardid string) *Card {
  if card := trello.cardById[cardid]; card == nil {
    data := Card{ trello: trello, Id: cardid, Members: NewSet() }
    data.update()
    data.cache()
    return &data
  } else {
    return card
  }
}

/* Attach issues, PRs and commits to the card */
func (card *Card) attachURL(addr string) {
  GenPOSTForm(card.trello, "/cards/" + card.Id + "/attachments", nil, url.Values{ "url": { addr } })
  // TODO if error
}

func (card *Card) AttachIssue(issue *github.Issue) {
  card.attachURL(issue.IssueURL())
}

/* Move a card to the different list */
func (card *Card) Move(listid string) {
  log.Printf("Moving card %s to list %s.", card.Id, listid)
  GenPUT(card.trello, "/cards/" + card.Id + "/idList?value=" + listid)
}

/* Find card by Issue. Assuming only one such card exists. */
func (trello *Trello) FindCard(issue string) *Card {
  return trello.cardByIssue[issue]
}

/* Fetch all cards from the board and [re-]initialise caches */
func (trello *Trello) makeCardCache() {
  var data []Card
  GenGET(trello, "/boards/" + trello.BoardId + "/cards/", &data)

  for _, card := range data {
    card.trello = trello
    card.update()
    card.cache()
  }
}

/* Attach an Issue link */
func (card *Card) LinkIssue(issue *github.Issue) {
  if (card.issue != issue) {
    card.trello.cardByIssue[issue.String()] = card
    card.issue = issue
  }
}

/* TODO handlers:
 - card created
 - card links updated
 */
