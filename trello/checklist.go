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
  Items   []CheckItem            `json:"checkItems"`
  id2in   map[string]int         `json:"-"`
  in2id   []string               `json:"-"`
  card    *Card
}

/* Creates an empty checklist */
func (card *Card) NewChecklist() {
  card.Checklist = new(Checklist)
  card.Checklist.Items = nil
  card.Checklist.card = card
  card.Checklist.updateLookup()
}

/* Copy contructor */
func (card *Card) CopyChecklist(checklist *Checklist) {
  card.Checklist = new(Checklist)
  card.Checklist.card = card
  card.Checklist.Items = make([]CheckItem, len(checklist.Items))
  for i, v := range checklist.Items {
    card.Checklist.Items[i] = CheckItem{ v.State == "completed", v.Text, v.Id, v.State }
  }
  card.Checklist.Id = checklist.Id
  card.Checklist.updateLookup()
}

/* Updates the lookup tables */
func (checklist *Checklist) updateLookup() {
  checklist.id2in = make(map[string]int)
  if n := len(checklist.Items); n > 0 {
    checklist.in2id = make([]string, n)
    for i,v := range checklist.Items {
      checklist.in2id[i] = v.Id
      checklist.id2in[v.Id] = i
    }
  }
}

/* Add a checklist to the card and return the id */
func (card *Card) AddChecklist() *Checklist {
  log.Printf("Adding a checklist to the card %s.", card.Id)
  card.NewChecklist()
  GenPOSTForm(card.trello, "/cards/" + card.Id + "/checklists", card.Checklist, url.Values{})

  return card.Checklist;
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
func (checklist *Checklist) UpdateItemName(i int, newname string) {
  log.Printf("Updating checklist item %d with new name %s.", i, newname)
  GenPUT(checklist.card.trello, "/cards/" + checklist.card.Id + "/checklist/" + checklist.Id +
    "/checkItem/" + checklist.in2id[i] + "/name?value=" + url.QueryEscape(newname))
}

func (checklist *Checklist) UpdateItemState(i int, newstate bool) {
  statestr := "incomplete"
  if newstate {
    statestr = "complete"
  }
  log.Printf("Updating checklist item %d with new state %s.", i, statestr)
  GenPUT(checklist.card.trello, "/cards/" + checklist.card.Id + "/checklist/" + checklist.Id +
    "/checkItem/" + checklist.in2id[i] + "/state?value=" + url.QueryEscape(statestr))
}

/* Remove a checkitem */
func (checklist *Checklist) DelItem(i int) {
  log.Printf("Deleting checklist item %d.", i)
  GenDEL(checklist.card.trello, "/checklists/" + checklist.Id + "/checkItems/" + checklist.in2id[i])
}

/* Remove whole checklist */
func (card *Card) DelChecklist() {
  if card.Checklist != nil {
    GenDEL(card.trello, "/card/" + card.Id + "/checklists/" + card.Checklist.Id)
  }
}

/* Add an item to checklist, note must have an Id */
func (checklist *Checklist) AddToChecklist(itm CheckItem) {
  n := len(checklist.Items) 
  checklist.Items = append(checklist.Items, itm)
  checklist.in2id = append(checklist.in2id, itm.Id)  
  checklist.id2in[itm.Id] = n
}

/* Unlink an item from the checklist */
func (checklist *Checklist) Unlink(i int) {
  //log.Printf("%v %v %v", checklist.Items, checklist.in2id, checklist.id2in)
  delete(checklist.id2in, checklist.in2id[i])
  if i < len(checklist.Items) - 1 {
    checklist.in2id = append(checklist.in2id[:i], checklist.in2id[i+1:]...)
    checklist.Items = append(checklist.Items[:i], checklist.Items[i+1:]...)
  } else {
    checklist.in2id = checklist.in2id[:i]
    checklist.Items = checklist.Items[:i]
  }
  /* Shift the indices after the deleted element */
  for j := i; j < len(checklist.in2id); j++ {
    checklist.id2in[checklist.in2id[j]] = j
  }
  //log.Printf("%v %v %v", checklist.Items, checklist.in2id, checklist.id2in)
}

/* Wrapper around the map */
func (checklist *Checklist) At(id string) int {
  v, present := checklist.id2in[id]
  if !present {
    return -1 
  } else {
    return v
  }
}

/* Loads the first checklist from Trello */
func (card *Card) LoadChecklists() {
  var data []Checklist
  GenGET(card.trello, "/cards/" + card.Id + "/checklists", &data)

  if len(data) > 0 {
    card.CopyChecklist(&data[0])
  } else {
    card.Checklist = nil
  }
}

/* Renders the checklist into GitHub's Markdown */
func tick(checked bool) byte {
  if checked {
    return 'X'
  } else { 
    return ' ' 
  }
}

func (checklist *Checklist) Render() string {
  res := ""
  for _, v := range checklist.Items {
    res = res + fmt.Sprintf("\r\n- [%c] %s", tick(v.Checked), v.Text)
  }
  return res
}
