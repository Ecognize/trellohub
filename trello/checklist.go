/* Operations with Trello checklists */
package trello

import (
  . "github.com/ErintLabs/trellohub/genapi"
  "log"
  "net/url"
)

type Checklist struct {
  Object
  state   []CheckItem
  card    *Card
}

/* Add a checklist to the card and return the id */
func (card *Card) addChecklist() *Checklist {
  log.Printf("Adding a checklist to the card %s.", card.Id)
  card.checklist = new(Checklist)
  card.checklist.card = card
  GenPOSTForm(card.trello, "/cards/" + card.Id + "/checklists", &card.checklist, url.Values{})

  return card.checklist;
}

/* Add an item to the checklist */
func (checklist *Checklist) postToCheckList(itm CheckItem) {
  log.Printf("Adding checklist item: %s.", itm.Text)
  var checkedTxt string
  if itm.Checked {
    checkedTxt = "true"
  } else {
    checkedTxt = "false"
  }
  GenPOSTForm(checklist.card.trello, "/checklists/" + checklist.Id + "/checkItems", nil,
    url.Values{ "name": { itm.Text }, "checked": { checkedTxt } })
}

/* Synchronise the list with incoming information from GitHub */
func (card *Card) UpdateChecklist(itms []CheckItem) {
  if card.checklist == nil {
    card.addChecklist()
  }
  // TODO handle edit events merging the lists
  for _, v := range itms {
    card.checklist.postToCheckList(v)
  }
  // TODO put this on update!
  card.checklist.state = itms
}
