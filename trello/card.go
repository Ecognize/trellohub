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
  Object
  ListId      string      `json:"idList"`
  Desc        string      `json:"desc"`
  members     []string
  trello      *Trello
  issue       *github.Issue
  checklist   *Checlist
}

/* Places the card in the cache */
func (card *Card) cache() {
  card.trello.cardById[card.id] = card
}

/* Updates card data from the server */
func (card *Card) update() {
  GenGET(card.trello, "/cards/" + card.Id, card)
  // TODO if error

  /* We don't really care to hold attachments array, just check if there is something to link */
  var data []Object
  GenGET(trello, "/cards/" + card.Id + "/attachments/", &data)
  issuesFound := 0
  for i := range data {
    re := regexp.MustCompile(REGEX_GH_ISSUE)
    if re.FindStringSubmatch(event.Action.Data.Attach.URL); res != nil {
      issuesFound ++;
      if issuesFound > 1 {
        log.Printf("WARNING: Duplicate issue attachments found on card #%s.", card.Id)
      }

      issueno, _ := strconv.Atoi(res[2])
      card.LinkIssue(card.trello.github.GetIssue(res[1], issueno))
    }
  }

  // TODO fetch users

  // REFACTOR: checklist
}

/* Adds a card to the list with a given name and returns the card id */
func (trello *Trello) AddCard(listid string, name string, desc string) string {
  data := Card{}
  GenPOSTForm(trello, "/cards/", &data, url.Values{
    "name": { name },
    "idList": { listid },
    "desc": { desc },
    "pos": { "top" } })

  card.cache()
  // TODO if error

  return data.id
}

/* Retrieves the card from the server */
func (trello *Trello) GetCard(cardid string) *Card {
  if card := trello.cardById[cardid]; card == nil {
    data := Card{ trello: trello, id: cardid }
    data.update()
    data.cache()
    return &data
  } else {
    return card
}

/* Attach issues, PRs and commits to the card */
func (card *Card) attachURL(addr string) {
  GenPOSTForm(card.trello, "/cards/" + card.id + "/attachments", nil, url.Values{ "url": { addr } })
  // TODO if error
}

func (card *Card) AttachIssue(issue *github.Issue) {
  card.attachURL(issue.IssueURL())
}

/* Move a card to the different list */
func (card *Card) Move(listid string) {
  log.Printf("Moving card %s to list %s.", card.id, listid)
  GenPUT(trello, "/cards/" + card.id + "/idList?value=" + listid)
}

/* Find card by Issue. Assuming only one such card exists. */
func (trello *Trello) FindCard(issue string) *Card {
  return trello.cardByIssue[issue]
}

/* Fetch all cards from the board and [re-]initialise caches */
func (trello *Trello) makeCardCache() {
  var data []Card
  GenGET(trello, "/boards/" + trello.BoardId + "/cards/", &data)

  for card := range data {
    card.update()
    card.cache()
  }
}

/* Handlers to model update */
func (card *Card) UpdateName(name string) {
  card.Name = name
}

func (card *Card) UpdateDesc(desc string) {
  card.Desc = desc
}

func (card *Card) LinkIssue(issue *github.Issue) {
  if (card.issue != issue) {
    card.issue = new(github.IssueSpec) // REFACTOR issues in GitHub
    trello.cardByIssue[*issue] = card
  }
}

/* TODO handlers:
 - card created
 - card links updated
 */
