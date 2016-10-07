/* Operations with GitHub labels */
package github

import (
  . "../genapi"
  "log"
  "strconv"
)

/* Adds a label to the issue */
func (issue *Issue) AddLabel(label string) {
  log.Printf("Adding label %s to %s", label, string(issue))
  lbls := [...]string { label }
  GenPOSTJSON(issue.github, issue.ApiURL() + "/labels", nil, &lbls)
}

/* Removes a label from the issue */
func (issue *Issue) DelLabel(label string) {
  log.Printf("Removing label %s from %s", label, string(issue))
  GenDEL(issue.github, issue.ApiURL() + "/labels/" + label) // TODO test if 404 would happen to crash us
}
