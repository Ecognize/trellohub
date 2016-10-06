/* Operations with Trello checklists */
package trello

import (
  . "../genapi"
  "log"
  "net/url"
)

/* Add a checklist to the card and return the id */
func (this *Trello) AddChecklist(cardid string) string {
  log.Printf("Adding a checklist to the card %s.", cardid)
  data := Object{}
  GenPOSTForm(this, "/cards/" + cardid + "/checklists", &data, url.Values{})

  return data.Id;
}

/* Add an item to the checklist */
func (this *Trello) AddToCheckList(checklistid string, itm CheckItem) {
  log.Printf("Adding checklist item: %s.", itm.Text)
  var checkedTxt string
  if itm.Checked {
    checkedTxt = "true"
  } else {
    checkedTxt = "false"
  }
  GenPOSTForm(this, "/checklists/" + checklistid + "/checkItems", nil,
    url.Values{ "name": { itm.Text }, "checked": { checkedTxt } })
}
