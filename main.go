package main

import (
    "fmt"
    "log"
    "net/http"
    "os"
    "io/ioutil"
    "encoding/json"
    "regexp"
    . "github.com/ErintLabs/trellohub/genapi"
    "github.com/ErintLabs/trellohub/trello"
    "github.com/ErintLabs/trellohub/github"
)

/* Globals are bad */
var trello_obj *trello.Trello
var github_obj *github.GitHub;
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
  GitLabelByListId    map[string]string
  ListIdByGitLabel    map[string]string
  GitHubUserByTrello       map[string]string
  TrelloUserByGitHub       map[string]string
}

func GetEnv(varname string) string {
  res := os.Getenv(varname)
  if len(res) <= 0 {
    log.Fatalf("$%s must be set.", varname)
  }

  return res
}

func main() {
  /* Check if we are run to [re]-initialise the board */
  if (len(os.Args) >= 4) {
    config.TrelloKey, config.TrelloToken, config.BoardId = os.Args[1], os.Args[2], os.Args[3]
    trello_obj = trello.New(config.TrelloKey, config.TrelloToken, config.BoardId)

    /* Archive all open lists */
    for _, v := range trello_obj.GetLists() {
      v.Close()
    }

    /* Ugly but effective, creating new lists */
    trello_obj.Lists = trello.ListRef{
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
    github_obj = github.New(config.GitHubToken)
    trello_obj.Startup(github_obj)

    /* List indexes */
    json.Unmarshal([]byte(GetEnv("LISTS")), &trello_obj.Lists)

    /* Trello to GitHub correspondence, also reversing */
    json.Unmarshal([]byte(GetEnv("USER_TABLE")), &cache.GitHubUserByTrello)
    cache.TrelloUserByGitHub = DicRev(cache.GitHubUserByTrello)

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
    cache.GitLabelByListId = map[string]string {
      trello_obj.Lists.InboxId: "inbox",
      trello_obj.Lists.InWorksId: "work",
      trello_obj.Lists.BlockedId: "block",
      trello_obj.Lists.ReviewId: "review",
      trello_obj.Lists.MergedId: "merged",
      trello_obj.Lists.DeployId: "deploy",
      trello_obj.Lists.TestId: "test",
      trello_obj.Lists.AcceptId: "done",
    }
    cache.ListIdByGitLabel = DicRev(cache.GitLabelByListId)

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
    var event trello.Payload
    json.Unmarshal(body, &event)
    evt := event.Action.Type

    /* Determining which action happened */
    switch (evt) {
    case "addAttachmentToCard":
      /* Check if the list is correct */
      card := trello_obj.GetCard(event.Action.Data.Card.Id)
      if card.ListId == trello_obj.Lists.ReposId {
        /* Check if this is a GitHub URL after all */
        re := regexp.MustCompile(REGEX_GH_REPO)
        if res := re.FindStringSubmatch(event.Action.Data.Attach.URL); res != nil {
          repoid := res[1]
          log.Printf("Registering new repository: %s.", repoid)

          /* Add a label, but make sure no duplicates happen */
          if trello_obj.GetLabel(repoid) == "" {
            card.SetLabel(trello_obj.AddLabel(repoid))
          } else {
            log.Print("Label already there, not proceeding.")
          }

          /* Installing webhooks if necessary */
          github_obj.EnsureHook(repoid, config.BaseURL)
        }
      } // TODO do we want to dance with other types of card attachments? e.g. somebody manually adds an issue link
      return http.StatusOK, "Attachment processed."
      // TODO: process removals and updates

    case "updateCard":
      card := trello_obj.GetCard(event.Action.Data.Card.Id)
      /* That's a big class of events, let's concentrate on what we want */
      if len(event.Action.Data.ListB.Id) > 0 && len(event.Action.Data.ListA.Id) > 0 {
        /* The card has been moved, check if it has an issue to it */
        oldlist := event.Action.Data.ListB.Id
        newlist := event.Action.Data.ListA.Id

        if card.Issue != nil && oldlist != newlist {
          /* Update labels if necessary */
          if label := cache.GitLabelByListId[oldlist]; len(label) > 0 {
            card.Issue.DelLabel(label)
          }
          if label := cache.GitLabelByListId[newlist]; len(label) > 0 {
            card.Issue.AddLabel(label)
          }
        }

        card.ListId = event.Action.Data.ListA.Id
      }
      // TODO:
      // - card rename
      // - card description update
      return http.StatusOK, "Card update processed."

    case "addMemberToCard", "removeMemberFromCard":
      card := trello_obj.GetCard(event.Action.Data.Card.Id)
      userid := event.Action.Data.Member
      add := evt[0] != 'r'
      card.Members[userid] = add

      /* Check that the user is in the table */
      if tuser := trello_obj.UserById(userid) ; len(tuser) > 0 {
        /* TODO: maybe generalise this process */
        if issue := card.Issue; issue != nil {
          guser := cache.GitHubUserByTrello[tuser] // assert len()>0
          present := issue.Members[guser]

          if (add && !present) {
            issue.AddUser(guser)
            return http.StatusOK, "User added."
          } else if (!add && present) {
            issue.DelUser(guser)
            return http.StatusOK, "User removed."
          }
        } else {
          return http.StatusOK, "No issue to the card, call the cops, I don't care."
        }
      } else {
        return http.StatusNotFound, "Sorry I have no idea who that user is."
      }

    case "addChecklistToCard", "createCheckItem",
      "updateCheckItemStateOnCard", "updateCheckItem",
      "deleteCheckItem":
      card := trello_obj.GetCard(event.Action.Data.Card.Id)
      /* If card has no issue, drop it */
      if card.Issue == nil {
        return http.StatusOK, "Not an issue card."
      }
      /* Run event dependent checks */
      switch (evt) {
      /* Only interested in first checklist */
      case "addChecklistToCard":
        if card.Checklist != nil {
          return http.StatusOK, "Got a checklist already."
        }
      case "createCheckItem", "updateCheckItemStateOnCard", "deleteCheckItem":
        if card.Checklist == nil || card.Checklist.Id != event.Action.Data.ChList.Id {
          return http.StatusOK, "Not interested in that checklist."
        }
        event.Action.Data.ChItem.FromTrello()
      }
      /* Update the model */
      switch (evt) {
      case "addChecklistToCard":
        checklist := new(trello.Checklist)
        *checklist = event.Action.Data.ChList
        card.LinkChecklist(checklist)
        // TODO: checklists which already have items!
        return http.StatusOK, "New checklist registered"
      case "createCheckItem":
        item := new(CheckItem)
        *item = event.Action.Data.ChItem
        card.Checklist.AddToChecklist(item)
      case "updateCheckItemStateOnCard":
        /* TODO assert not nil */
        card.Checklist.State[event.Action.Data.ChItem.Id].Checked = event.Action.Data.ChItem.Checked
      case "updateCheckItem":
        card.Checklist.State[event.Action.Data.ChItem.Id].Text = event.Action.Data.ChItem.Text
      case "deleteCheckItem":
        delete(card.Checklist.State, event.Action.Data.ChItem.Id)
      }
      /* Regenerate the new issue body and update it */
      card.Issue.UpdateBody(RepMentions(card.Desc, cache.GitHubUserByTrello) + card.Checklist.Render(cache.GitHubUserByTrello))
      return http.StatusOK, "Checklists updated"

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
    var payload github.Payload
    json.Unmarshal(body, &payload)

    /* Guess we have a new issue */
    switch (payload.Action) {
    case "opened":
      /* Look up the corresponding trello label */
      if labelid := trello_obj.GetLabel(payload.Repo.Spec); len(labelid) > 0 {
        /* Generating an in-DB refernce */
        issue := github_obj.GetIssue(payload.Repo.Spec, payload.Issue.IssueNo)
        newbody, checkitems := issue.GetChecklist(cache.TrelloUserByGitHub, payload.Issue.Body)

        /* Insert the card, attach the issue and label */
        card := trello_obj.AddCard(trello_obj.Lists.InboxId, payload.Issue.Title, newbody)
        card.AttachIssue(issue)
        card.SetLabel(labelid)

        issue.AddLabel("inbox")
        issue.SetLabels(payload.Issue.LabelsDb)
        issue.SetMembers(payload.Issue.Assigs)
        for k, v := range issue.Members {
          if v {
            card.AddUser(cache.TrelloUserByGitHub[k])
          }
        }

        /* Form a checklist */
        if len(checkitems) > 0 {
          card.UpdateChecklist(checkitems)
        }

        /* Happily report */
        log.Printf("Creating card %s for issue %s\n", card.Id, issue.String())
        return http.StatusOK, "Got your back, captain."
      } else {
        return http.StatusNotFound, "You sure we serve this repo? I don't think so."
      }

    case "labeled","unlabeled":
      issue := github_obj.GetIssue(payload.Repo.Spec, payload.Issue.IssueNo)
      label := payload.Label.Name
      add := payload.Action[0] !='u'
      issue.Labels[label] = add

      if listid, card := cache.ListIdByGitLabel[label], trello_obj.FindCard(issue.String());
        add && len(listid) > 0 && card != nil {
        /* If the card is not in that list already, request the move */
        if curlist := card.ListId; curlist != listid {
          card.Move(listid)
          return http.StatusOK, "Understood, moving card."
        } else {
          return http.StatusOK, "The card was already there but thank you."
        }
      } else if card == nil {
        return http.StatusNotFound, "Can't find a corresponding card, probably it was created before we started serving this repo."
      }

    case "assigned", "unassigned":
      issue := github_obj.GetIssue(payload.Repo.Spec, payload.Issue.IssueNo)
      user := payload.Assignee.Name
      add := payload.Action[0] !='u'
      issue.Members[user] = add

      /* Find the card and the user */
      if tuser, card := cache.TrelloUserByGitHub[user], trello_obj.FindCard(issue.String());
        len(tuser) > 0 && len(card.Id) > 0 {
        /* Determine mode of operation */
        present := card.Members[trello_obj.UserByName(tuser)]

        /* Check if the user is already assigned there, to prevent WebAPI recursion */
        if (add && !present) || (!add && present)  {
          if (add) {
            card.AddUser(tuser)
          } else {
            card.DelUser(tuser)
          }
          return http.StatusOK, "Card users updated."
        } else {
          return http.StatusOK, "Well I already know this anyway."
        }
      /* Something's wrong */
      } else {
        if len(tuser) <= 0 {
          return http.StatusNotFound, "We do not serve user" + user + "."
        } else {
          return http.StatusNotFound, "Can't find the corresponding card, probably issue is older than sync."
        }
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
