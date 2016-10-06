/* Operations with GitHub issues */
package github

import (
  . "../genapi"
  "strconv"
  "log"
)

type Issue struct {
   RepoId   string
   IssueNo  int
}

/* Auto-converions to string */
func (IssueSpec this) genconv(middlepart string) string {
  return this.RepoId + middlepart + strconv.Itoa(this.IssueNo)
}

func (IssueSpec this) String() string {
  return this.genconv("#")
}

func (IssueSpec this) IssueURL() string {
  return "https://github.com/" + this.genconv("/issues/")
}

/* Check if a user is assigned to the card */
func (this *GitHub) UsersAssigned(issue IssueSpec) []string {
  /* Strangely we can reuse it here */
  var payload Payload
  GenGET(this, "repos/" + issue.RepoId + "/issues/" + strconv.Itoa(issue.IssueNo), &payload)

  /* Populate a string slice */
  res := make([]string, len(payload.Assigs))
  for i, v := range payload.Assigs {
    res[i] = v.Name
  }

  return res
}

type userAssignRequest struct {
  Assigs  []string `json:"assignees"`
}

/* Assign/Unassign a user to the card */
func (this *GitHub) ReassignUsers(users []string, issue IssueSpec) {
  payload := userAssignRequest{users}

  GenPCHJSON(this, "repos/" + issue.RepoId + "/issues/" + strconv.Itoa(issue.IssueNo), &payload)
  log.Printf("Issue %s#%d assignees update.", issue.RepoId, issue.IssueNo)
}
