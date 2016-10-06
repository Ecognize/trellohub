/* Operations with GitHub labels */
package github

import (
  . "../genapi"
  "log"
  "strconv"
)

type labelSpec struct {
  Name  string  `json:"name"`
}

/* Check if the issue has a specific label attached to it */
// TODO: cache
func (this *GitHub) HasLabel(issue IssueSpec, label string) bool {
  var oldldbs []labelSpec
  GenGET(this, "repos/" + issue.rid + "/issues/" + strconv.Itoa(issue.iid) + "/labels", &oldldbs)

  for _, v := range oldldbs {
    if v.Name == label {
      return true
    }
  }

  return false
}

/* Adds a label to the issue */
func (this *GitHub) AddLabel(issue IssueSpec, label string) {
  /* Checking if the label isn't there yet to prevent Trello-GitHub recursion */
  if !this.HasLabel(issue, label) {
    log.Printf("Adding label %s to %s#%d", label, issue.rid, issue.iid)
    lbls := [...]string { label }
    GenPOSTJSON(this, "repos/" + issue.rid + "/issues/" + strconv.Itoa(issue.iid) + "/labels", nil, &lbls)
  }
}

/* Removes a label from the issue */
func (this *GitHub) DelLabel(issue IssueSpec, label string) {
  /* Checking if the label is present actually */
  if this.HasLabel(issue, label) {
    log.Printf("Removing label %s from %s#%d", label, issue.rid, issue.iid)
    GenDEL(this, "repos/" + issue.rid + "/issues/" + strconv.Itoa(issue.iid) + "/labels/" + label)
  }
}
