/* Operations with Trello checklists */
package trello

import (
  . "github.com/ErintLabs/trellohub/genapi"
  "log"
  "net/url"
  "fmt"
)

type Checklist struct {
  Id      string                 `json:"id"`
  Items   []CheckItem            `json:"checkItems"` // only used at load!
  State   map[string]*CheckItem  `json:"-"`
  card    *Card
}

/* Add a checklist to the card and return the id */
func (card *Card) AddChecklist() *Checklist {
  log.Printf("Adding a checklist to the card %s.", card.Id)
  card.Checklist = new(Checklist)
  card.Checklist.card = card
  card.Checklist.State = make(map[string]*CheckItem)
  GenPOSTForm(card.trello, "/cards/" + card.Id + "/checklists", card.Checklist, url.Values{})

  return card.Checklist;
}

func (card *Card) LinkChecklist(checklist *Checklist) {
  card.Checklist = checklist
  card.Checklist.card = card
  card.Checklist.State = make(map[string]*CheckItem)
}

/* Add an item to the checklist and returns id */
func (checklist *Checklist) PostToChecklist(itm CheckItem) string {
  log.Printf("Adding checklist item: %s.", itm.Text)
  var checkedTxt string
  if itm.Checked {
    checkedTxt = "true"
  } else {
    checkedTxt = "false"
  }
  var data Object
  GenPOSTForm(checklist.card.trello, "/checklists/" + checklist.Id + "/checkItems", &data,
    url.Values{ "name": { itm.Text }, "checked": { checkedTxt } })
  return data.Id
}

/* Updates an item state */
func (checklist *Checklist) UpdateItemName(itemid string, newname string) {
  log.Printf("Updating item %s with new name %s.", itemid, newname)
  GenPUT(checklist.card.trello, "/cards/" + checklist.card.Id + "/checklists/" + checklist.Id +
    "/checkItems/" + itemid + "/name?value=" + url.QueryEscape(newname))
}

func (checklist *Checklist) UpdateItemState(itemid string, newstate bool) {
  statestr := "incomplete"
  if newstate {
    statestr = "complete"
  }
  log.Printf("Updating item %s with new state %s.", itemid, statestr)
  GenPUT(checklist.card.trello, "/cards/" + checklist.card.Id + "/checklists/" + checklist.Id +
    "/checkItems/" + itemid + "/state?value=" + url.QueryEscape(statestr))
}

/* Remove a checkitem */
func (checklist *Checklist) DelItem(itemid string) {
  GenDEL(checklist.card.trello, "/checklists/" + checklist.Id + "/checkItems/" + itemid)
}

/* Remove whole checklist */
func (card *Card) DelChecklist() {
  if card.Checklist != nil {
    GenDEL(card.trello, "/card/" + card.Id + "/checklists/" + card.Checklist.Id)
  }
}

/* Add an item to checklist */
func (checklist *Checklist) AddToChecklist(itm *CheckItem) {
  checklist.State[itm.Id] = itm
}

/* Loads the first checklist from Trello */
func (card *Card) LoadChecklists() {
  var data []Checklist
  GenGET(card.trello, "/cards/" + card.Id + "/checklists", &data)

  if len(data) > 0 {
    checklist := new(Checklist)
    checklist.Id = data[0].Id
    card.LinkChecklist(checklist)
    for _, v := range data[0].Items {
      itm := new(CheckItem)
      *itm = v
      itm.FromTrello()
      checklist.AddToChecklist(itm)
    }
  } else {
    card.Checklist = nil
  }
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
