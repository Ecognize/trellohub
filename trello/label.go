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
func (this *Trello) SetLabel(cardid string, labelid string) {
    GenPOSTForm(this, "/cards/" + cardid + "/idLabels", nil, url.Values{ "value": { labelid } })
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
func (this *Trello) GetLabel(repoid string) string {
  /* Look in cache, if not there retry */
  for updated := false; !updated; updated = this.makeLabelCache() {
    if id, ok := this.labelCache[repoid]; ok {
      return id
    }
  }

  /* If we are still there, something's wrong */
  return ""
}

/* Looks up a label to corresponding repository, returns an empty string if not found */
func (this *Trello) FindLabel(addr string) string {
  /* Break the incoming string down to just Owner/repo */
  var key string
  re := regexp.MustCompile(REGEX_GH_REPO)
  if res := re.FindStringSubmatch(addr); res != nil {
    key = res[2] + "/" + res[3]
  } else {
    log.Fatal("Incoming URL fails GitHubness, what's going on?")
    return ""
  }

  return this.GetLabel(key)
}
