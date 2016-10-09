/* Operations with GitHub labels */
package github

import (
  . "../genapi"
  "log"
)

type Label struct {
  Name      string    `json:"name"`
}

type GitUser  struct {
  Name      string  `json:"login"`
}

func (issue *Issue) SetLabels(lbls []Label) {
  lst := make([]string, len(lbls))
  for i, v := range lbls {
    lst[i] = v.Name
  }
  issue.Labels.SetNameable(lst)
}

func (issue *Issue) SetMembers(mbmrs []GitUser) {
  lst := make([]string, len(mbmrs))
  for i, v := range mbmrs {
    lst[i] = v.Name
  }
  issue.Members.SetNameable(lst)
}

/* Adds a label to the issue */
func (issue *Issue) AddLabel(label string) {
  log.Printf("Adding label %s to %s", label, issue.String())
  lbls := [...]string { label }
  GenPOSTJSON(issue.github, issue.ApiURL() + "/labels", nil, &lbls)
}

/* Removes a label from the issue */
func (issue *Issue) DelLabel(label string) {
  log.Printf("Removing label %s from %s", label, issue.String())
  GenDEL(issue.github, issue.ApiURL() + "/labels/" + label) // TODO test if 404 would happen to crash us
}

/* Adds a user to the issue */
type userAssignRequest struct {
  Assigs  []string `json:"assignees"`
}

func (issue *Issue) AddUser(user string) {
  payload := userAssignRequest{ []string{ user } }
  GenPOSTJSON(issue.github, issue.ApiURL() + "/assignees", nil, &payload)
  log.Printf("Added user %s to issue %s.", user, issue.String())
}

/* Removes a use from the issue */
func (issue *Issue) DelUser(user string) {
  payload := userAssignRequest{ []string{ user } }
  GenDELJSON(issue.github, issue.ApiURL() + "/assignees", &payload)
  log.Printf("Removed user %s from issue %s.", user, issue.String())
}
