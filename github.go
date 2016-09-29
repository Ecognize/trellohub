package main

import (
  "regexp"
  "strconv"
)

type GitHub struct {
  Token     string
}

func NewGitHub(token string) *GitHub {
  t := new(GitHub)
  t.Token = token

  return t
}

type GitCommit struct {
  Commit    struct {
    Message string  `json:"message"`
  }                 `json:"commit"`
}

func (this *GitHub) AuthQuery() string {
  return "access_token=" + this.Token
}

func (this *GitHub) BaseURL() string {
  return "https://api.github.com/"
}

type IssueSpec struct {
   rid string
   iid int
}

func extractIssueIds(message string, repoid string) []IssueSpec {
  res := make([]IssueSpec, 0)

  /* Assuming no single commit can close more than 256 issues okay */
  re := regexp.MustCompile("(?i)(?:close|closes|closed|fix|fixes|fixed|resolve|resolves|resolved)[[:space:]]*([a-z0-9][a-z0-9-]{0,38}[a-z0-9]/[a-z0-9][a-z0-9-]{0,38}[a-z0-9])?#([0-9]*)")
  if catch := re.FindAllStringSubmatch(message, 256); catch != nil {
    for _, v := range catch {
      /* Check if there was a repo specification before # */
      var repo string
      if len(v[1]) > 0 {
        repo = v[1]
      } else {
        repo = repoid
      }

      /* Add the new cath */
      iid, _ := strconv.Atoi(v[2])
      res = append(res, IssueSpec{ repo, iid } )
    }
  }

  return res
}

/* List ids of issues affected by a PR */
func (this *GitHub) AffectedIssues(pr IssueSpec) []IssueSpec {
  res := make([]IssueSpec, 0)

  /* Fetching commit data for the PR */
  var commits []GitCommit
  GenGET(this, "repos/" + pr.rid + "/pulls/" + strconv.Itoa(pr.iid) + "/commits", &commits)

  /* Parsing messages and finding relevant issues */
  for _, v := range commits {
    if issues := extractIssueIds(v.Commit.Message, pr.rid); len(issues) > 0 {
      res = append(res, issues...)
    }
  }

  return res
}

/* Check if the issue has a specific label attached to it */
func (this *GitHub) HasLabel(issue IssueSpec, label string) bool {
  var oldldbs []string
  GenGET(this, "repos/" + issue.rid + "/issues/" + strconv.Itoa(issue.iid) + "/labels", &oldldbs)

  for _, v := range oldldbs {
    if v == label {
      return true
    }
  }

  return false
}

/* Adds a label to the issue */
func (this *GitHub) AddLabel(issue IssueSpec, label string) {
  /* Checking if the label isn't there yet to prevent Trello-GitHub recursion */
  if !this.HasLabel(issue, label) {
    lbls := [...]string { label }
    GenPOSTJSON(this, "repos/" + issue.rid + "/issues/" + strconv.Itoa(issue.iid) + "/labels", nil, &lbls)
  }
}

/* Removes a label from the issue */
func (this *GitHub) DelLabel(issue IssueSpec, label string) {
  /* Checking if the label is present actually */
  if this.HasLabel(issue, label) {
    GenDEL(this, "repos/" + issue.rid + "/issues/" + strconv.Itoa(issue.iid) + "/labels/" + label)
  }
}
