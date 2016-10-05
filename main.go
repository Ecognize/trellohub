package main

import (
    "fmt"
    "log"
    "net/http"
    "os"
    "io/ioutil"
    "encoding/json"
    "regexp"
    "strconv"
)

type checkItem struct {
  Checked bool
  Text    string
}

/* TODO: move to GitHub */
type IssuePayload struct {
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

/* TODO: move to Trello */
type TrelloPayload struct {
  Action      struct {
    Type      string        `json:"type"`
    Data      struct {
      Member  string        `json:"idMember"`
      List    TrelloObject  `json:"list"`
      Card    TrelloObject  `json:"card"`
      ListB   TrelloObject  `json:"listBefore"`
      ListA   TrelloObject  `json:"listAfter"`
      Attach  struct {
        URL   string        `json:"url"`
      }                     `json:"attachment"`
    }                       `json:"data"`
  }                         `json:"action"`
}

/* Globals are bad */
var trello *Trello
var github *GitHub;
var config struct {
  BaseURL       string
  BoardId       string
  TrelloKey     string
  TrelloToken   string
  GitHubToken   string
  Port          string
  StableBranch  string
  TestBranch    string
}

// TODO noncaputre group
const REGEX_GH_REPO string = "^(https?://)?github.com/([^/]*)/([^/]*)"

// TODO: this ignores nesting, only top level is processed
// TODO: this might not work well with backslashes
const REGEX_GH_CHECK string = "(?:^|\\r\\n)- \\[([ x])\\] ([^\\r]*)"

// TODO: possibly separate GH and Trello version
const REGEX_GH_USER string = "(?i)@([a-z0-9][a-z0-9-]{0,38}[a-z0-9])"
var cache struct {
  ListLabels    map[string]string
  LabelLists    map[string]string
  Trello2GitHub map[string]string
  GitHub2Trello map[string]string
}

/* Reverse a dictionary (check if standar exist?) */
func DicRev(dic map[string]string) map[string]string {
  res := make(map[string]string)
  for k, v := range dic {
    res[v] = k
  }
  return res
}

func GetEnv(varname string) string {
  res := os.Getenv(varname)
  if len(res) <= 0 {
    log.Fatalf("$%s must be set.", varname)
  }

  return res
}

/* Lua style strsub, replaces all matches of a regexp with what the callback returns
   Doesn't edit the original string */
// TODO handle bogus regexps
type strsub_c func (v []string) string
func strsub(source string, regtxt string, f strsub_c) string {
  res := source
  re := regexp.MustCompile(regtxt)
  j := 0

  for {
    catch := re.FindStringSubmatchIndex(res[j:])
    if catch == nil {
      break
    }

  	// TODO proper slices maybe
  	lc := len(catch)/2
  	lo := catch[1] - catch[0]
    par := make([]string, lc)
    for i := 0; i < lc; i++ {
      par[i] = res[catch[i*2] + j:catch[i*2+1] + j]
    }
    rep := f(par)
    res = res[:catch[0] + j] + rep + res[catch[1] + j:]
    j = catch[1] + j - lo + len(rep)
  }
  return res
}

/* Replaces all occurences of @mentions between GitHub and Trello
   second parameter determines the dictionary */
func repMentions(text string, dic map[string]string) string {
  return strsub(text, REGEX_GH_USER, func (v []string) string {
    return "@"+dic[v[1]]
  })
}

func main() {
  /* Check if we are run to [re]-initialise the board */
  if (len(os.Args) >= 4) {
    config.TrelloKey, config.TrelloToken, config.BoardId = os.Args[1], os.Args[2], os.Args[3]
    trello = NewTrello(config.TrelloKey, config.TrelloToken, config.BoardId)

    /* Archive all open lists */
    for _, v := range trello.ListIds() {
      trello.CloseList(v)
    }

    /* Ugly but effective, creating new lists */
    trello.Lists = ListRef{
      trello.AddList("📋 Repositories"),
      trello.AddList("📥 Inbox"),
      trello.AddList("🚧 In Works"),
      trello.AddList("🚫 Blocked"),
      trello.AddList("📝 Awaiting Review"),
      trello.AddList("💾 Merged to Mainline"),
      trello.AddList("📲 Deployed on Test"),
      trello.AddList("📱 Tested"),
      trello.AddList("📤 Accepted"),
    }

    /* Happily print the JSON */
    data, _ := json.Marshal(trello.Lists)
    fmt.Println("Set $LISTS to the following value:")
    fmt.Println(string(data[:]))
  } else {
    /* Server configuration */
    config.BaseURL, config.Port = GetEnv("URL"), GetEnv("PORT")

    /* Trello config */
    config.TrelloKey, config.TrelloToken = GetEnv("TRELLO_KEY"), GetEnv("TRELLO_TOKEN")
    config.BoardId = GetEnv("BOARD")

    /* GitHub config */
    config.GitHubToken = GetEnv("GITHUB_TOKEN")
    config.StableBranch, config.TestBranch = GetEnv("STABLE_BRANCH"), GetEnv("TEST_BRANCH")

    /* Instantiating globals */
    trello = NewTrello(config.TrelloKey, config.TrelloToken, config.BoardId)
    github = NewGitHub(config.GitHubToken)

    /* List indexes */
    json.Unmarshal([]byte(GetEnv("LISTS")), &trello.Lists)

    /* Trello to GitHub correspondence, also reversing */
    json.Unmarshal([]byte(GetEnv("USER_TABLE")), &cache.Trello2GitHub)
    cache.GitHub2Trello = DicRev(cache.Trello2GitHub)

    /* Registering handlers */
    http.HandleFunc("/trello", TrelloFunc)
    http.HandleFunc("/trello/", TrelloFunc)

    http.HandleFunc("/issues", IssuesFunc)
    http.HandleFunc("/issues/", IssuesFunc)

    http.HandleFunc("/pull", PullFunc)
    http.HandleFunc("/pull/", PullFunc)

    /* Ensuring Trello hook */
    /* TODO: study if this doesn't cause races */
    // TODO: ex SIGTERM problem
    go trello.EnsureHook(config.BaseURL + "/trello")

    /* Fill the GitHub label names cache */
    cache.ListLabels = map[string]string {
      trello.Lists.InboxId: "inbox",
      trello.Lists.InWorksId: "work",
      trello.Lists.BlockedId: "block",
      trello.Lists.ReviewId: "review",
      trello.Lists.MergedId: "merged",
      trello.Lists.DeployId: "deploy",
      trello.Lists.TestId: "test",
      trello.Lists.AcceptId: "done",
    }
    cache.LabelLists = DicRev(cache.ListLabels)

    /* Starting the server up */
    log.Fatal(http.ListenAndServe(":"+config.Port, nil))
  }
}

type handleSubroutine func (body []byte) (int, string)

func GeneralisedProcess(w http.ResponseWriter, r *http.Request, f handleSubroutine) {
  // TODO io.LimitReader
  // TODO check if its or POST
  body, err := ioutil.ReadAll(r.Body)
  if err != nil {
      log.Fatal(err)
  }

  /* Invoking the actual function */
  //log.Print(string(body[:]))
  var code int
  var text string

  if r.Method != "HEAD" {
    code, text = f(body)
  } else { /* or not, if it's a HEAD */
    code, text = http.StatusOK, "Pleased to meet you."
  }

  /* Replying to the caller */
  w.WriteHeader(code)
  fmt.Fprintln(w, text)

  /* Finalise session */
  if err := r.Body.Close(); err != nil {
      log.Fatal(err)
  }
}

func TrelloFunc(w http.ResponseWriter, r *http.Request) {
  GeneralisedProcess(w, r, func (body []byte) (int, string) {
    event := TrelloPayload{}
    json.Unmarshal(body, &event)

    /* Determining which action happened */
    switch (event.Action.Type) {
    case "addAttachmentToCard":
      /* Check if the list is correct */
      if trello.CardList(event.Action.Data.Card.Id) == trello.Lists.ReposId {
        /* Check if this is a GitHub URL after all */
        re := regexp.MustCompile(REGEX_GH_REPO)
        if res := re.FindStringSubmatch(event.Action.Data.Attach.URL); res != nil {
          repoid := res[2] + "/" + res[3]
          log.Printf("Registering new repository: %s.", repoid)

          /* Add a label, but make sure no duplicates happen */
          if trello.GetLabel(repoid) == "" {
            trello.SetLabel(event.Action.Data.Card.Id, trello.AddLabel(repoid))
          } else {
            log.Print("Label already there, not proceeding.")
          }

          /* Installing webhooks if necessary */
          github.EnsureHook(repoid, config.BaseURL)
        }
      }
      return http.StatusOK, "Attachment processed."

    case "updateCard":
      /* That's a big class of events, let's concentrate on what we want */
      if len(event.Action.Data.ListB.Id) > 0 && len(event.Action.Data.ListA.Id) > 0 {
        /* The card has been moved, check if it has a repo */
        re := regexp.MustCompile(REGEX_GH_REPO + "/issues/([0-9]*)")
        if res := re.FindStringSubmatch(trello.FirstLink(event.Action.Data.Card.Id)); res != nil {
          // TODO cache
          issue := IssueSpec{ res[2] + "/" + res[3], 0 }
          issue.iid, _ = strconv.Atoi(res[4])

          /* Remove the label if necessary */
          if label := cache.ListLabels[event.Action.Data.ListB.Id]; len(label) > 0 {
              github.DelLabel(issue, label)
          }

          /* Add the label if necessary */
          if label := cache.ListLabels[event.Action.Data.ListA.Id]; len(label) > 0 {
              github.AddLabel(issue, label)
          }

          return http.StatusOK, "New labels adjusted."
        }
      }

    case "addMemberToCard", "removeMemberFromCard":
      /* Check that the user is in the table */
      if tuser := trello.UserById(event.Action.Data.Member); len(tuser) > 0 {
        /* TODO: maybe generalise this process */
        /* TODO: cache! */
        re := regexp.MustCompile(REGEX_GH_REPO + "/issues/([0-9]*)")
        if res := re.FindStringSubmatch(trello.FirstLink(event.Action.Data.Card.Id)); res != nil {
          // TODO cache
          issue := IssueSpec{ res[2] + "/" + res[3], 0 }
          issue.iid, _ = strconv.Atoi(res[4])
          guser := cache.Trello2GitHub[tuser] // assert len()>0

          users := github.UsersAssigned(issue)
          assign, user_idx := event.Action.Type[0] != 'r', -1
          for i, v := range users {
            if v == guser {
              user_idx = i
              break
            }
          }

          // TODO GH cache
          if (assign && user_idx < 0) || (!assign && user_idx >= 0)  {
            if (assign) {
              users = append(users, guser)
            } else {
              /* Evil magic from golang wiki: remove an element w/o order preservation */
              users[user_idx] = users[len(users) - 1]
              users = users[:len(users) - 1]
            }
            github.ReassignUsers(users, issue)
            return http.StatusOK, "Issue users updated."
          } else {
            return http.StatusOK, "I knew that already."
          }
        } else {
          return http.StatusOK, "No issue to the card, call the cops, I don't care."
        }
      } else {
        return http.StatusNotFound, "Sorry I have no idea who that user is."
      }

    default:
      log.Print(string(body[:]))
    }

    return http.StatusOK, "Erm, hello."
  })
}

func IssuesFunc(w http.ResponseWriter, r *http.Request) {
  GeneralisedProcess(w, r, func (body []byte) (int, string) {
    /* TODO check json errors */
    /* TODO check it was github who sent it anyway */
    /* TODO check whether we serve this repo */
    var issue IssuePayload
    json.Unmarshal(body, &issue)

    /* Guess we have a new issue */
    switch (issue.Action) {
    case "opened":
      /* Look up the corresponding trello label */
      if labelid := trello.FindLabel(issue.Issue.URL); len(labelid) > 0 {
        /* TODO: reuse this for updates */
        checkitems := make([]checkItem, 0)
        newbody := strsub(issue.Issue.Body, REGEX_GH_CHECK, func (v []string) string {
          checkitems = append(checkitems, checkItem{ v[1][0] != ' ', repMentions(v[2], cache.GitHub2Trello) })
          return ""
        })
        /* Separated for readability */
        newbody = repMentions(newbody, cache.GitHub2Trello)

        /* Insert the card, attach the issue and label */
        cardid := trello.AddCard(trello.Lists.InboxId, issue.Issue.Title, newbody)
        trello.AttachURL(cardid, issue.Issue.URL)
        trello.SetLabel(cardid, labelid)
        github.AddLabel(IssueSpec{issue.Repo.Spec, issue.Issue.Number}, "inbox")

        /* Form a checklist */
        if len(checkitems) > 0 {
          checkid := trello.AddChecklist(cardid)
          for _, v := range checkitems {
            trello.AddToCheckList(checkid, v)
          }
        }

        /* Happily report */
        log.Printf("Creating card %s for issue %s\n", cardid, issue.Issue.URL)
        return http.StatusOK, "Got your back, captain."
      } else {
        return http.StatusNotFound, "You sure we serve this repo? I don't think so."
      }

    case "labeled":
      /* Check if the label is one that we serve and there is a card for the issue */
      if listid, cardid := cache.LabelLists[issue.Label.Name], trello.FindCard(IssueSpec{issue.Repo.Spec, issue.Issue.Number});
        len(listid) > 0 && len(cardid) > 0 {
        /* If the card is not in that list already, request the move */
        if curlist := trello.CardList(cardid); curlist != listid {
          trello.MoveCard(cardid, listid)
          return http.StatusOK, "Understood, moving card."
        } else {
          return http.StatusOK, "The card was already there but thank you."
        }
      } else if len(cardid) <= 0 {
        return http.StatusNotFound, "Can't find a corresponding card, probably it was created before we started serving this repo."
      }


    case "assigned", "unassigned":
      /* Find the card and the user */
      if tuser, cardid := cache.GitHub2Trello[issue.Assignee.Name], trello.FindCard(IssueSpec{issue.Repo.Spec, issue.Issue.Number});
        len(tuser) > 0 && len(cardid) > 0 {
        /* Determine mode of operation */
        assign, user_there := issue.Action[0] != 'u', trello.UserAssigned(tuser, cardid)

        /* Check if the user is already assigned there, to prevent WebAPI recursion */
        if (assign && !user_there) || (!assign && user_there)  {
          if (assign) {
            trello.AssignUser(tuser, cardid)
          } else {
            trello.UnassignUser(tuser, cardid)
          }
          return http.StatusOK, "Card users updated."
        } else {
          return http.StatusOK, "Well I already know this anyway."
        }
      /* Something's wrong */
      } else {
        return http.StatusNotFound, "Either this user is not one of us, or the card is nowhere to be found. I dunno man."
      }
    }

    return http.StatusOK, "I can't really process this, but fine."
  })
}

func PullFunc(w http.ResponseWriter, r *http.Request) {
  GeneralisedProcess(w, r, func (body []byte) (int, string) {
    log.Print(string(body[:]))

    return http.StatusOK, "I can't really process this, but fine."
  })
}
