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

/* Parse the commit message and try to find issues it closes */
type issueSpec struct { repo string; id int } 
func extractIssueIds(message string, repoid string) []issueSpec {
  res := make([]issueSpec, 0)
  
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
      res = append(res, issueSpec{ repo, iid } )
    }
  }
  
  return res
}

/* List ids of issues affected by a PR */
func (this *GitHub) AffectedIssues(repoid string, prid int) []issueSpec {
  res := make([]issueSpec, 0)
  
  /* Fetching commit data for the PR */
  var commits []GitCommit
  GenGET(this, "repos/" + repoid + "/pulls/" + strconv.Itoa(prid) + "/commits", &commits)
  
  /* Parsing messages and finding relevant issues */
  for _, v := range commits {
    if issues := extractIssueIds(v.Commit.Message, repoid); len(issues) > 0 {
      res = append(res, issues...)
    }
  }
  
  return res
}
