package github

import (
  "log"
  . "../genapi"
)

type Payload struct {
  Action  string    `json:"action"`
  Issue struct {
    URL   string    `json:"html_url"`
    Title string    `json:"title"`
    Body  string    `json:"body"`
    Number  int     `json:"number"`
  }                 `json:"issue"`
  Repo  struct {
    Spec  string    `json:"full_name"`
  }                 `json:"repository"`
  Assignee GitUser  `json:"assignee"` // TODO deprecated field!
  Assigs []GitUser  `json:"assignees"`
  Label struct {
    Name  string    `json:"name"`
  }                 `json:"label"`
}

type GitHub struct {
  Token     string
}

/* Make some fields private maybe */
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

type IssueSpec struct {
   RepoId   string
   IssueNo  int
}

func New(token string) *GitHub {
  t := new(GitHub)
  t.Token = token

  return t
}

func (this *GitHub) AuthQuery() string {
  return "access_token=" + this.Token
}

func (this *GitHub) BaseURL() string {
  return "https://api.github.com/"
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

