/* Operations with GitHub issues */
package github

import (
  . "../genapi"
  "strconv"
)

type Issue struct {
  RepoId    string
  URL       string    `json:"html_url"`
  Title     string    `json:"title"`
  Body      string    `json:"body"`
  IssueNo   int       `json:"number"`
  LabelsDb  []Label   `json:"labels"`
  Assignees
  github    *GitHub

  Labels    Set
  Members   Set
}

/* Auto-converions to string */
func (issue *Issue) genconv(middlepart string) string {
  return issue.RepoId + middlepart + strconv.Itoa(issue.IssueNo)
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

/* Places the issue in the lookup cache and creates */
func (issue *Issue) cache() {
  issue.github.issueBySpec[issue.String()] = issue
}

/* Retrieves the issue data from the server */
func (issue *Issue) update() {
  GenGET(issue.github, issue.ApiURL(), issue)
}

/* Parses body and outputs the actual body and checklists, takes username correspondence table as input */
// TODO nested checklists (#24)
func (issue *Issue) GetChecklist(utable map[string]string, body string) (string, []CheckItem) {
  checkitems := make([]CheckItem, 0)
  res := StrSub(body, REGEX_GH_CHECK, func (v []string) string {
    checkitems = append(checkitems, CheckItem{ v[1][0] != ' ', RepMentions(v[2], utable) })
    return ""
  })

  return RepMentions(res, utable), checkitems
}

/* Requests a reference to the issue */
func (github *GitHub) GetIssue(repoid string, issueno int) *Issue {
  res := Issue{ RepoId: repoid, IssueNo: issueno}
  if issue := github.issueBySpec[res.String()]; issue != nil {
    return issue
  } else {
    res.github = github
  //  res.update() Do we need it ever?
    res.cache()
    res.Members = NewSet()
    res.Labels = NewSet()
    return &res
  }
}
