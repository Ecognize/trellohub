/* Operations with Trello checklists */
package trello

import (
  . "github.com/ErintLabs/trellohub/genapi"
  "log"
  "net/url"
  "fmt"
)

type Checklist struct {
  Object
  State   map[string]*CheckItem  `json:"-"`
  card    *Card
}

/* Add a checklist to the card and return the id */
func (card *Card) addChecklist() *Checklist {
  log.Printf("Adding a checklist to the card %s.", card.Id)
  card.Checklist = new(Checklist)
  card.Checklist.card = card
  card.Checklist.State = make(map[string]*CheckItem)
  GenPOSTForm(card.trello, "/cards/" + card.Id + "/checklists", &card.Checklist, url.Values{})

  return card.Checklist;
}

func (card *Card) LinkChecklist(checklist *Checklist) {
  card.Checklist = checklist
  card.Checklist.card = card
  card.Checklist.State = make(map[string]*CheckItem)
}

/* Add an item to the checklist */
func (checklist *Checklist) postToChecklist(itm CheckItem) {
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
  if card.Checklist == nil {
    card.addChecklist()
  }
  // TODO handle edit events merging the lists
  for _, v := range itms {
    card.Checklist.postToChecklist(v)
  }
  // TODO put this on update!
  //card.Checklist.State = itms
}

/* Add an item to checklist */
func (checklist *Checklist) AddToChecklist(itm CheckItem) {
  checklist.State[itm.Id] = &itm
}

/* Renders the checklist into GitHub's Markdown */
func (checklist *Checklist) Render(table map[string]string) string {
  res := ""
  for _, v := range checklist.State {
    var on byte
    if (v.Checked) {
      on = 'X'
    } else {
      on = ' '
    }
    res = res + fmt.Sprintf("\r\n- [%c] %s", on, v.Text)
  }
  return res
}
