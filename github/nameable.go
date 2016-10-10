/* Operations with GitHub labels */
package github

import (
  . "github.com/ErintLabs/trellohub/genapi"
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

/* TODO: maybe unify this all over after all */

/* Adds a label to the issue */
func (issue *Issue) AddLabel(label string) {
  log.Printf("Adding label %s to %s", label, issue.String())
  lbls := [...]string { label }
  GenPOSTJSON(issue.github, issue.ApiURL() + "/labels", nil, &lbls)
}

/* Removes a label from the issue */
func (issue *Issue) DelLabel(label string) {
  log.Printf("Removing label %s from %s", label, issue.String())
  GenDEL(issue.github, issue.ApiURL() + "/labels/" + label) // TODO ensure 403/404 doesn't crash us
}

/* Adds a user to the issue */
type userAssignRequest struct {
  Assigs  []string `json:"assignees"`
}

func (issue *Issue) AddUser(user string) {
  log.Printf("Adding user %s to %s", user, issue.String())
  payload := userAssignRequest{ []string{ user } }
  GenPOSTJSON(issue.github, issue.ApiURL() + "/assignees", nil, &payload)
}

/* Removes a use from the issue */
func (issue *Issue) DelUser(user string) {
  log.Printf("Removing user %s from %s", user, issue.String())
  payload := userAssignRequest{ []string{ user } }
  GenDELJSON(issue.github, issue.ApiURL() + "/assignees", &payload)
}
