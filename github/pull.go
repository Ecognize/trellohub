/* Operations with GitHub rull requests */
package github

import (
  . "../genapi"
  "strconv"
  "regexp"
)


func extractIssueIds(message string, repoid string) []IssueSpec {
  res := make([]IssueSpec, 0)

  /* Assuming no single commit can close more than 256 issues okay */
  re := regexp.MustCompile(REGEX_GH_MAGIC)
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
