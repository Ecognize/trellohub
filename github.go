package main

import (
  "regexp"
  "strconv"
  "log"
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

type GitUser  struct {
  Name      string  `json:"login"`
}

type WebHook struct {
  Name      string    `json:"name"`
  Active    bool      `json:"active"`
  Events    []string  `json:"events"`
  Config    struct {
    Type    string    `json:"content_type"`
    URL     string    `json:"url"`
  }                   `json:"config"`
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

/* Check and install webhooks on a repository */
// TODO secret
// TODO don't fail if we don't have access
func (this *GitHub) EnsureHook(repoid string, callbackURLbase string) {
  /* Retrieving previously installed hooks */
  var hooks []WebHook
  GenGET(this, "repos/" + repoid + "/hooks", &hooks)

  hookevts := map[string] struct { event string; found bool } {
    "/issues": { "issues", false },
    "/pull": { "pull_request", false},
  }

  /* Checking if there is a hook with exact same parameters */
  for _, v := range hooks {
    for k, f := range hookevts {
      if len(v.Events) > 0 && v.Events[0] == f.event && v.Config.URL == callbackURLbase + k {
        log.Printf("Found an existing GitHub hook at %s for %s, reusing.", v.Config.URL, repoid)
        hookevts[k] = struct{event string; found bool}{ f.event, true }
      }
    }
  }

  /* Creating hooks that failed */
  for k, f := range hookevts {
    if !f.found {
      /* TODO Compound initialisation? */
      var wh WebHook
      wh.Name = "web"
      wh.Active = true
      wh.Events = []string{ f.event }
      wh.Config.Type = "json"
      wh.Config.URL = callbackURLbase + k
      log.Printf("Creating a hook for %s at %s", wh.Config.URL, repoid)
      GenPOSTJSON(this, "repos/" + repoid + "/hooks", nil, &wh)
    }
  }
}

/* Check if a user is assigned to the card */
func (this *GitHub) UsersAssigned(issue IssueSpec) []string {
  /* Strangely we can reuse it here */
  var payload IssuePayload
  GenGET(this, "repos/" + issue.rid + "/issues/" + strconv.Itoa(issue.iid), &payload)

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

  GenPCHJSON(this, "repos/" + issue.rid + "/issues/" + strconv.Itoa(issue.iid), &payload)
  log.Printf("Issue %s#%d assignees update.", issue.rid, issue.iid)
}
