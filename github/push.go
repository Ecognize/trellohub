/* Operations with GitHub rull requests */
package github

import (
 . "github.com/ErintLabs/trellohub/genapi"
 "strconv"
 "regexp"
)

type Push struct {
  Commits []Commit  `json:"commits"`
  Repo    Repo      `json:"repository"`
}

/* List ids of issues affected by a PR */
func (push *Push) AffectedIssues() []*Issue {
  res := make([]*Issue, 0)

  /* Parsing messages and finding relevant issues */
  for _, v := range push.Commits {
    if issues := pull.github.extractIssueIds(v.Message, pull.RepoId); len(issues) > 0 {
      res = append(res, issues...)
    }
  }

  return res
}
