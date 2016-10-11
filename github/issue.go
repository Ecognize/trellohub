/* Operations with GitHub issues */
package github

import (
  . "github.com/ErintLabs/trellohub/genapi"
  "strconv"
)

type Issue struct {
  RepoId      string          `json:"-"`
  URL         string          `json:"html_url"`
  Title       string          `json:"title"`
  Body        string          `json:"body"`
  IssueNo     int             `json:"number"`
  LabelsDb    []Label         `json:"labels"`
  Assignees
  github      *GitHub

  Labels      Set             `json:"-"`
  Members     Set             `json:"-"`
  Checkmap    map[string]int  `json:"-"`
  Checklist   []CheckItem     `json:"-"`
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
  newbody, checkitems := issue.GetChecklist(cache.TrelloUserByGitHub, payload.Issue.Body)
}

/* Parses body and outputs the actual body and checklists, takes username correspondence table as input */
// TODO nested checklists (#24)
func (issue *Issue) GetChecklist(utable map[string]string, body string) (string, []CheckItem) {
  checkitems := make([]CheckItem, 0)
  res := StrSub(body, REGEX_GH_CHECK, func (v []string) string {
    checkitems = append(checkitems, CheckItem{ Checked: v[1][0] != ' ', Text: RepMentions(v[2], utable) })
    return ""
  })

  return RepMentions(res, utable), checkitems
}

/* Requests a reference to the issue */
func (github *GitHub) GetIssue(repoid string, issueno int) *Issue {
  res := &Issue{ RepoId: repoid, IssueNo: issueno}
  if issue := github.issueBySpec[res.String()]; issue != nil {
    return issue
  } else {
    res.github = github
    res.update() Do we need it ever?
    res.cache()
    res.Members = NewSet()
    res.Labels = NewSet()
    return res
  }
}

/* Updates Issue body/title */
func (issue *Issue) UpdateBody(newbody string) {
  GenPATCHJSON(issue.github, issue.ApiURL(), &struct { Body string `json:"body"` }{ newbody })
}

func (issue *Issue) UpdateTitle(newtitle string) {
  GenPATCHJSON(issue.github, issue.ApiURL(), &struct { Title string `json:"title"` }{ newtitle })
}
