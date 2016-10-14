/* Operations with GitHub rull requests */
package github

import (
 . "github.com/ErintLabs/trellohub/genapi"
 "strconv"
 "regexp"
)

type Pull Issue

func (github *GitHub) extractIssueIds(message string, repoid string) []*Issue {
  res := make([]*Issue, 0)

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
      res = append(res, github.GetIssue(repo, iid))
    }
  }

  return res
}


/* List ids of issues affected by a PR */
func (pull *Pull) AffectedIssues() []*Issue {
  res := make([]*Issue, 0)

  /* Fetching commit data for the PR */
  var commits []GitCommit
  GenGET(pull.github, pull.ApiURL() + "/commits", &commits)

  /* Parsing messages and finding relevant issues */
  for _, v := range commits {
    if issues := pull.github.extractIssueIds(v.Commit.Message, pull.RepoId); len(issues) > 0 {
      res = append(res, issues...)
    }
  }

  return res
}

/* Requests a reference to the pr */
func (github *GitHub) GetPull(repoid string, issueno int) *Pull {
  res := &Pull{ RepoId: repoid, IssueNo: issueno}
  if pull := github.pullBySpec[res.String()]; pull != nil {
    return pull
  } else {
    res.github = github
    res.update() 
    res.cache()
    res.Members = NewSet()
    res.Labels = NewSet()
    return res
  }
}

/* Auto-converions to string */
// TODO maybe an interface w/ Issue? or merge into one class
func (pull *Pull) genconv(middlepart string) string {
  return pull.RepoId + middlepart + strconv.Itoa(pull.IssueNo)
}

func (pull *Pull) String() string {
  return pull.genconv("#")
}

func (pull *Pull) IssueURL() string {
  return "https://github.com/" + pull.genconv("/pulls/")
}

func (pull *Pull) ApiURL() string {
  return "repos/" + pull.genconv("/pulls/")
}


/* Places the issue in the lookup cache and creates */
func (pull *Pull) cache() {
  pull.github.pullBySpec[pull.String()] = pull
}

/* Retrieves the issue data from the server */
func (pull *Pull) update() {
  GenGET(pull.github, pull.ApiURL(), pull)
}
