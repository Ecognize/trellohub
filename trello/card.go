/* Operations with Trello cards */
package trello

import (
  . "github.com/ErintLabs/trellohub/genapi"
  "github.com/ErintLabs/trellohub/github"
  "net/url"
  "log"
  "strconv"
  "regexp"
  . "time" // what the fug? without direct import it complains
)

type Card struct {
  // Object TODO cascading
  Id          string        `json:"id"`
  Name        string        `json:"name"`
  ListId      string        `json:"idList"`
  Desc        string        `json:"desc"`
  trello      *Trello
  Issue       *github.Issue `json:"-"`
  Checklist   *Checklist    `json:"-"`
  Members     Set           `json:"-"`
}

/* Places the card in the cache */
func (card *Card) cache() {
  card.trello.cardById[card.Id] = card
  if card.Issue != nil {
    issuestr := card.Issue.String()
    if card.trello.cardByIssue[issuestr] != card {
      log.Printf("Card %s registered for issue %s", card.Id, issuestr)
      card.trello.cardByIssue[issuestr] = card
    }
  }
}

/* Updates card data from the server */
func (card *Card) load() {
  GenGET(card.trello, "/cards/" + card.Id, card)
  // TODO if error

  /* We don't really care to hold attachments array, just check if there is something to link */
  var data []Object
  GenGET(card.trello, "/cards/" + card.Id + "/attachments", &data)
  issuesFound := 0
  for _, v := range data {
    log.Printf("Found attachment: %s", v.Name)
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

  card.loadMembers()
  card.LoadChecklists()
}

/* Adds a card to the list with a given name and returns the card id */
func (trello *Trello) AddCard(listid string, name string, desc string) *Card {
  data := &Card{ trello: trello, Members: NewSet() }
  GenPOSTForm(trello, "/cards/", data, url.Values{
    "name": { name },
    "idList": { listid },
    "desc": { desc },
    "pos": { "top" } })

  data.cache()
  // TODO if error

  return data
}

/* Retrieves the card from the server */
func (trello *Trello) GetCard(cardid string) *Card {
  if card := trello.cardById[cardid]; card == nil {
    data := &Card{ trello: trello, Id: cardid, Members: NewSet() }
    data.load()
    data.cache()
    return data
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
  /* We don't have a change to wait until the update, add up instantly */
  card.Issue = issue
  card.cache()
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
  GenGET(trello, "/boards/" + trello.BoardId + "/cards", &data)

  for _, v := range data {
    card := new(Card)
    *card = v
    card.trello = trello
    card.load()
    card.cache()

    // TODO implement a more sophisticated rate limit evasion mechanism
    // for now it's ok
    log.Printf("Sleeping 1 second")
    Sleep(1 * Second)
  }
}

/* Attach an Issue link */
func (card *Card) LinkIssue(issue *github.Issue) {
  if (card.Issue != issue) {
    card.Issue = issue
    card.cache()
  }
}

/* Update name/description */
func (card *Card) UpdateName(newname string) {
  GenPUT(card.trello, "/cards/" + card.Id + "/name?value=" + url.QueryEscape(newname))
}

func (card *Card) UpdateDesc(newdesc string) {
  GenPUT(card.trello, "/cards/" + card.Id + "/desc?value=" + url.QueryEscape(newdesc))
}

/* TODO handlers:
 - card created
 - card links updated
 */
