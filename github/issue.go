/* Operations with GitHub issues */
package github

import (
  . "../genapi"
  "strconv"
  "log"
)

type Issue struct {
  RepoId    string
  URL       string    `json:"html_url"`
  Title     string    `json:"title"`
  Body      string    `json:"body"`
  IssueNo   int       `json:"number"`
  github    *GitHub
}

/* Auto-converions to string */
func (issue *Issue) genconv(middlepart string) string {
  return issue.Repo() + middlepart + strconv.Itoa(issue.IssueNo)
}

func (issue *Issue) String() string {
  return issue.genconv("#")
}

func (issue *Issue) IssueURL() string {
  return "https://github.com/" + issue.genconv("/issues/")
}

func (issue *Issue) ApiURL() string {
  return "repos/" + issue.genconv("/issues/")
}

/* If the RepoId field is missing then we probably got this issue over the wire,
   cut the correct part from the HTML url */
func (issue *Issue) Repo() string {
  if len(issue.RepoId) <= 0 {
    re := regexp.MustCompile(REGEX_GH_REPO)
    if res := re.FindStringSubmatch(issue.URL); res != nil {
      issue.RepoId = res[1]
    } else {
      Log.Fatalf("URL %s is not a GitHub repository, what is happening here?")
    }
  }
  return issue.RepoId
}

/* Sets the global GitHub object reference and auto-register */
func (issue *Issue) SetGitHub(github *GitHub) {
  issue.github = github
  issue.cache()
}

/* Places the issue in the lookup cache */
func (issue *Issue) cache() {
  issue.github.issueBySpec[*issue] = issue
}

/* Retrieves the issue data from the server */
func (issue *Issue) update() {
  GenGET(issue.github, issue.ApiURL(), issue)
}

/* Parses body and outputs the actual body and checklists, takes username correspondence table as input */
// TODO nested checklists (#24)
func (issue *Issue) GetChecklist(utable map[string]string) string, []trello.CheckItem {
  checkitems := make([]trello.CheckItem, 0)
  res := StrSub(issue.Issue.Body, REGEX_GH_CHECK, func (v []string) string {
    checkitems = append(checkitems, trello.CheckItem{ v[1][0] != ' ', repMentions(v[2], utable) })
    return ""
  })

  return repMentions(res, utable), checkitems
}

/* Requests a reference to the issue */
func (github *GitHub) GetIssue(repoid string, issueno int) *Issue {
  res := Issue{ RepoId: repoid, IssueNo: issueno}
  if issue := github.issueBySpec[string(res)]; issue != nil {
    return issue
  } else {
    res.github = github
    res.update()
    res.cache()
    return &res
  }
}


/* Assign/Unassign a user to the card */
type userAssignRequest struct {
  Assigs  []string `json:"assignees"`
}
func (issue *Issue) ReassignUsers(users []string) {
  payload := userAssignRequest{users}

  GenPCHJSON(this, issue.ApiURL(), &payload)
  log.Printf("Issue %s assignees update.", string(issue))
} // REFACTOR: add/remove maybe?
