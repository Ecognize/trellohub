/* Operations with GitHub rull requests */
package github

import (
  "regexp"
  . "github.com/ErintLabs/trellohub/genapi"
)

type Push struct {
  Ref     string    `json:"ref"`
  Branch  string    `json:"-"`
  Commits []Commit  `json:"commits"`
  Repo    Repo      `json:"repository"`
  Head    Commit    `json:"head_commit"`
  github  *GitHub   `json:"-"`
}

/* Sets the instance reference also parses the ref */
func (push *Push) SetGitHub(github *GitHub) {
  push.github = github

  re := regexp.MustCompile(REGEX_GH_BRANCH)
  if res := re.FindStringSubmatch(push.Ref); res != nil {
    push.Branch = res[1]
  }

}

/* List ids of issues affected by a PR */
func (push *Push) AffectedIssues() []*Issue {
  res := make([]*Issue, 0)

  /* Parsing messages and finding relevant issues */
  for _, v := range append(push.Commits, push.Head) {
    if issues := push.github.extractIssueIds(v.Message, push.Repo.Spec); len(issues) > 0 {
      res = append(res, issues...)
    }
  }

  return res
}
