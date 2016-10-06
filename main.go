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
    "./trello/trello"
)

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

/* Globals are bad */
var trello_obj *Trello
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

var cache struct {
  ListLabels    map[string]string
  LabelLists    map[string]string
  Trello2GitHub map[string]string
  GitHub2Trello map[string]string
}

func GetEnv(varname string) string {
  res := os.Getenv(varname)
  if len(res) <= 0 {
    log.Fatalf("$%s must be set.", varname)
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
    trello_obj = trello.New(config.TrelloKey, config.TrelloToken, config.BoardId)

    /* Archive all open lists */
    for _, v := range trello_obj.ListIds() {
      trello_obj.CloseList(v)
    }

    /* Ugly but effective, creating new lists */
    trello_obj.Lists = ListRef{
      trello_obj.AddList("📋 Repositories"),
      trello_obj.AddList("📥 Inbox"),
      trello_obj.AddList("🚧 In Works"),
      trello_obj.AddList("🚫 Blocked"),
      trello_obj.AddList("📝 Awaiting Review"),
      trello_obj.AddList("💾 Merged to Mainline"),
      trello_obj.AddList("📲 Deployed on Test"),
      trello_obj.AddList("📱 Tested"),
      trello_obj.AddList("📤 Accepted"),
    }

    /* Happily print the JSON */
    data, _ := json.Marshal(trello_obj.Lists)
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
    trello_obj = trello.New(config.TrelloKey, config.TrelloToken, config.BoardId)
    github = NewGitHub(config.GitHubToken)

    /* List indexes */
    json.Unmarshal([]byte(GetEnv("LISTS")), &trello_obj.Lists)

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
    go trello_obj.EnsureHook(config.BaseURL + "/trello")

    /* Fill the GitHub label names cache */
    cache.ListLabels = map[string]string {
      trello_obj.Lists.InboxId: "inbox",
      trello_obj.Lists.InWorksId: "work",
      trello_obj.Lists.BlockedId: "block",
      trello_obj.Lists.ReviewId: "review",
      trello_obj.Lists.MergedId: "merged",
      trello_obj.Lists.DeployId: "deploy",
      trello_obj.Lists.TestId: "test",
      trello_obj.Lists.AcceptId: "done",
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
    event := trello.Payload{}
    json.Unmarshal(body, &event)

    /* Determining which action happened */
    switch (event.Action.Type) {
    case "addAttachmentToCard":
      /* Check if the list is correct */
      if trello_obj.CardList(event.Action.Data.Card.Id) == trello_obj.Lists.ReposId {
        /* Check if this is a GitHub URL after all */
        re := regexp.MustCompile(REGEX_GH_REPO)
        if res := re.FindStringSubmatch(event.Action.Data.Attach.URL); res != nil {
          repoid := res[2] + "/" + res[3]
          log.Printf("Registering new repository: %s.", repoid)

          /* Add a label, but make sure no duplicates happen */
          if trello_obj.GetLabel(repoid) == "" {
            trello_obj.SetLabel(event.Action.Data.Card.Id, trello_obj.AddLabel(repoid))
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
        if res := re.FindStringSubmatch(trello_obj.FirstLink(event.Action.Data.Card.Id)); res != nil {
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
      if tuser := trello_obj.UserById(event.Action.Data.Member); len(tuser) > 0 {
        /* TODO: maybe generalise this process */
        /* TODO: cache! */
        re := regexp.MustCompile(REGEX_GH_REPO + "/issues/([0-9]*)")
        if res := re.FindStringSubmatch(trello_obj.FirstLink(event.Action.Data.Card.Id)); res != nil {
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
      if labelid := trello_obj.FindLabel(issue.Issue.URL); len(labelid) > 0 {
        /* TODO: reuse this for updates */
        checkitems := make([]checkItem, 0)
        newbody := strsub(issue.Issue.Body, REGEX_GH_CHECK, func (v []string) string {
          checkitems = append(checkitems, checkItem{ v[1][0] != ' ', repMentions(v[2], cache.GitHub2Trello) })
          return ""
        })
        /* Separated for readability */
        newbody = repMentions(newbody, cache.GitHub2Trello)

        /* Insert the card, attach the issue and label */
        cardid := trello_obj.AddCard(trello_obj.Lists.InboxId, issue.Issue.Title, newbody)
        trello_obj.AttachURL(cardid, issue.Issue.URL)
        trello_obj.SetLabel(cardid, labelid)
        github.AddLabel(IssueSpec{issue.Repo.Spec, issue.Issue.Number}, "inbox")

        /* Form a checklist */
        if len(checkitems) > 0 {
          checkid := trello_obj.AddChecklist(cardid)
          for _, v := range checkitems {
            trello_obj.AddToCheckList(checkid, v)
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
      if listid, cardid := cache.LabelLists[issue.Label.Name], trello_obj.FindCard(IssueSpec{issue.Repo.Spec, issue.Issue.Number});
        len(listid) > 0 && len(cardid) > 0 {
        /* If the card is not in that list already, request the move */
        if curlist := trello_obj.CardList(cardid); curlist != listid {
          trello_obj.MoveCard(cardid, listid)
          return http.StatusOK, "Understood, moving card."
        } else {
          return http.StatusOK, "The card was already there but thank you."
        }
      } else if len(cardid) <= 0 {
        return http.StatusNotFound, "Can't find a corresponding card, probably it was created before we started serving this repo."
      }


    case "assigned", "unassigned":
      /* Find the card and the user */
      if tuser, cardid := cache.GitHub2Trello[issue.Assignee.Name], trello_obj.FindCard(IssueSpec{issue.Repo.Spec, issue.Issue.Number});
        len(tuser) > 0 && len(cardid) > 0 {
        /* Determine mode of operation */
        assign, user_there := issue.Action[0] != 'u', trello_obj.UserAssigned(tuser, cardid)

        /* Check if the user is already assigned there, to prevent WebAPI recursion */
        if (assign && !user_there) || (!assign && user_there)  {
          if (assign) {
            trello_obj.AssignUser(tuser, cardid)
          } else {
            trello_obj.UnassignUser(tuser, cardid)
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
