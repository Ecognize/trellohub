/* Operations with Trello board labels */
package trello

import (
  . "../genapi"
  "log"
  "regexp"
  "net/url"
)

/* Add a label to board */
func (this *Trello) AddLabel(name string) string {
  /* Pick up a color first */
  colors := [...]string { "green", "yellow", "orange", "red", "purple", "blue", "sky", "lime", "pink", "black" }

  var labels []Object
  GenGET(this, "/boards/" + this.BoardId + "/labels/", &labels)

  /* TODO: avoid duplicates too */

  /* Create a label with appropriate color */
  col := colors[ (len(labels)-6) % len(colors) ]
  log.Printf("Creating a new %s label name %s in Trello.", col, name)
  data := Object{}
  GenPOSTForm(this, "/labels/", &data, url.Values{
    "name": { name },
    "idBoard": { this.BoardId },
    "color": { col } })

  this.labelCache[name] = data.Id

  return data.Id
}

/* Attach a label to the card */
func (card *Card) SetLabel(labelid string) {
    GenPOSTForm(this, "/cards/" + card.Id + "/idLabels", nil, url.Values{ "value": { labelid } })
}

/* Build a repo to label correspondence cache */
func (this *Trello) makeLabelCache() bool {
  var labels []Object
  GenGET(this, "/boards/" + this.BoardId + "/labels/", &labels)

  for _, v := range labels {
    this.labelCache[v.Name] = v.Id
  }

  return true // needed for dirty magic
}

/* Get the label id or empty string if not found */
func (trello *Trello) GetLabel(repoid string) string {
  // TODO monitor label add events instead of refreshing
  /* Look in cache, if not there retry */
  for updated := false; !updated; updated = trello.makeLabelCache() {
    if id, ok := trello.labelCache[repoid]; ok {
      return id
    }
  }

  /* If we are still there, something's wrong */
  return ""
}
