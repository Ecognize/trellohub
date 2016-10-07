/* Operations with GitHub labels */
package github

import (
  . "../genapi"
  "log"
  "strconv"
)

type Label struct {
  Name      string    `json:"name"`
}

type GitUser  struct {
  Name      string  `json:"login"`
}

type nameable interface {
  name() string
}

func (this *GitUser) name() string {
  return this.Name
}

func (this *Label) name() string {
  return this.Name
}

func (set *Set) setNameable(nm []nameable) {
  for k := range set {
    delete(set, k)
  }
  for _, v := range lbls {
    set[v.name()] = true
  }
}

func (issue *Issue) SetLabels(lbls []Label) {
  issue.Labels.setNameable(lbls)
}

func (issue *Issue) SetMembers(mbmrs []GitUser) {
  issue.Members.setNameable(mbmrs)
}

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

/* Adds a user to the issue */
type userAssignRequest struct {
  Assigs  []string `json:"assignees"`
}

func (issue *Issue) AddUser(user string) {
  payload := userAssignRequest{ { user } }
  GenPOSTJSON(issue.github, issue.ApiURL() + "/assignees", &payload)
  log.Printf("Added user %s to issue %s.", user, string(issue))
}

/* Removes a use from the issue */
func (issue *Issue) DelUser(user string) {
  payload := userAssignRequest{ { user } }
  GenDELJSON(issue.github, issue.ApiURL() + "/assignees", &payload)
  log.Printf("Removed user %s from issue %s.", user, string(issue))
}
