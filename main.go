package main

import (
    "fmt"
    "log"
    "net/http"
    "os"
    "io/ioutil"
    "encoding/json"
    "regexp"
    "sync"
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
  GitHubUserByTrello  map[string]string
  TrelloUserByGitHub  map[string]string
  mutex               sync.Mutex
}

func GetEnv(varname string) string {
  res := os.Getenv(varname)
  if len(res) <= 0 {
    log.Fatalf("$%s must be set.", varname)
  }

  return res
}

func g2t(str string) string {
  return RepMentions(str, cache.GitHubUserByTrello)
}

func t2g(str string) string {
  return RepMentions(str, cache.TrelloUserByGitHub)
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
  /* We don't care about performance, therefore enforce that only one proc can be running at a given time */
  cache.mutex.Lock()
  defer cache.mutex.Unlock()
  
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
    log.Printf("[Trello] %s", evt)

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
      /* If description changed */
      if event.Action.Data.Card.Desc != event.Action.Data.Old.Desc {
        card.Desc = event.Action.Data.Card.Desc
        /* Compare to the save one and regenerate if needed */
        if card.Issue != nil && g2t(card.Issue.Body) != card.Desc {
          newbody := card.Desc
          if card.Checklist != nil {
            newbody = newbody + card.Checklist.Render()
          }
          card.Issue.UpdateBody(t2g(newbody))
        }
      }
      /* If name changed */
      if event.Action.Data.Card.Name != event.Action.Data.Old.Name {
        card.Name = event.Action.Data.Card.Name
        /* Compare to the save one and update if needed */
        if card.Issue != nil && g2t(card.Issue.Title) != card.Name {
          card.Issue.UpdateTitle(t2g(card.Name))
        }
      }
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
      "deleteCheckItem", "removeChecklistFromCard":
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
      case "createCheckItem", "updateCheckItemStateOnCard", "deleteCheckItem", "removeChecklistFromCard":
        if card.Checklist == nil || card.Checklist.Id != event.Action.Data.ChList.Id {
          return http.StatusOK, "Not interested in that checklist."
        }
      }
      /* Sanity checks */
      switch (evt) {
      case "updateCheckItemStateOnCard", "deleteCheckItem":
        if card.Issue.Checklist == nil || card.Checklist == nil {
          log.Printf("[ERROR] Operating on nil checklist. issue.chlist = %#v card.chlist = %#v", card.Issue.Checklist, card.Checklist)
          return http.StatusInternalServerError, "Nil checklist encountered"
        }
      }
      /* Update the model */
      needsUpdate := true
      switch (evt) {
      case "addChecklistToCard":
        card.CopyChecklist(&event.Action.Data.ChList)
        /* If the checklist is empty, no need to update the issue */
        // TODO check if it works with pre-filled checklists
        if len(card.Checklist.Items) == 0 {
          return http.StatusOK, "New checklist registered"
        }
      case "createCheckItem":
        card.Checklist.AddToChecklist(event.Action.Data.ChItem)
        card.Checklist.EnsureOrder()
        if checklist := card.Issue.Checklist; checklist != nil && len(card.Checklist.Items) <= len(checklist) {
          needsUpdate = false
        }
      case "updateCheckItemStateOnCard":
        no := card.Checklist.At(event.Action.Data.ChItem.Id)
        check := event.Action.Data.ChItem.State == "complete"
        if card.Issue.Checklist[no].Checked == check {
          needsUpdate = false
        }
        card.Checklist.Items[no].Checked = check
      case "updateCheckItem":
        no := card.Checklist.At(event.Action.Data.ChItem.Id)
        if card.Issue.Checklist[no].Text == t2g(event.Action.Data.ChItem.Text) {
          needsUpdate = false
        }
        card.Checklist.Items[no].Text = event.Action.Data.ChItem.Text
      case "deleteCheckItem":
        /* If the lengths are the same, it's a Trello UI generated event */
        if len(card.Checklist.Items) != len(card.Issue.Checklist) {
          needsUpdate = false
        }
        card.Checklist.Unlink(card.Checklist.At(event.Action.Data.ChItem.Id))
      case "removeChecklistFromCard":
        card.Checklist = nil
        if card.Issue.Checklist == nil {
          needsUpdate = false
        }
      }
      if needsUpdate {
        /* Regenerate the new issue body and update it */
        newbody := card.Desc
        if card.Checklist != nil { /* We may have deleted the checklist */
          newbody = newbody + card.Checklist.Render()
        }
        card.Issue.UpdateBody(t2g(newbody))
        // TODO: remove when #32 is fixed
        card.Issue.Newbody = t2g(newbody)
      }
      return http.StatusOK, "Checklists updated"

    default:
      //log.Print(string(body[:]))
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
    log.Printf("[Github Issues] %s", payload.Action)
    
    /* Guess we have a new issue */
    switch (payload.Action) {
    case "opened","edited":
      /* Look up the corresponding trello label */
      if labelid := trello_obj.GetLabel(payload.Repo.Spec); len(labelid) > 0 {
        /* Generating an in-DB refernce and updating it */
        issue := github_obj.GetIssue(payload.Repo.Spec, payload.Issue.IssueNo)
        issue.Title = payload.Issue.Title
        
        // TODO: remove when #32 is fixed
        if payload.Changes.Body.From == payload.Issue.Body && len(issue.Newbody) > 0 {
          /* Aww crappity! */
          log.Printf("[BUG] Server sent us nonsense payload, using in-house data.")
          issue.Body = issue.Newbody
          issue.Newbody = payload.Issue.Body // just in case
        } else {        
          issue.Body = payload.Issue.Body
        }
        issue.GenChecklist()
        
        /* Shortcuts */
        trello_title := g2t(issue.Title)
        trello_descr := g2t(issue.Body)
        var card *trello.Card

        if payload.Action == "opened" {
          /* Insert the card, attach the issue and label */
          card = trello_obj.AddCard(trello_obj.Lists.InboxId, trello_title, trello_descr)
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

          /* Happily report */
          log.Printf("Creating card %s for issue %s\n", card.Id, issue.String())
        } else if payload.Action == "edited" {
          if card = trello_obj.FindCard(issue.String()); card != nil {
            /* Post updates to whichever attribute changed */
            if card.Name != trello_title {
              card.UpdateDesc(trello_title)
            }
            if card.Desc != trello_descr {
              card.UpdateDesc(trello_descr)
            }
          } else {
            return http.StatusNotFound, "Can't find the card, are we dealing with an old issue?"
          }
        }

        /* If issue is just opened or if it's an edit that might potentially add a checklist, try forming it */
        if payload.Action == "opened" || card.Checklist == nil {
          if len(issue.Checklist) > 0 {
            checklist := card.AddChecklist()
            for _, v := range issue.Checklist {
              checklist.PostToChecklist(CheckItem{ Text: g2t(v.Text) , Checked: v.Checked })
            }
          }
        } else if payload.Action == "edited" { /* Update the list */
          /* Corner case, user removed the list */
          if card.Checklist != nil && len(issue.Checklist) == 0 {
            card.DelChecklist()
          } else {
            /* Walk one by one and apply changes */
            for i, v := range issue.Checklist {
              /* If we overstep the original list means we have to add */
              if i >= len(card.Checklist.Items) {
                card.Checklist.PostToChecklist(v)
              } else { /* Otherwise post updates */
                if gtext := g2t(v.Text); gtext != card.Checklist.Items[i].Text {
                  card.Checklist.UpdateItemName(i, gtext)
                }
                if v.Checked != card.Checklist.Items[i].Checked {
                  card.Checklist.UpdateItemState(i, v.Checked)
                }
              }
            }
            /* If the incoming list was shorter, remove excess ones */
            for i := len(card.Checklist.Items) - 1 ; i >= len(issue.Checklist); i-- {
              card.Checklist.DelItem(i)
            }
          }
          /* TODO: some kind of merging algorithm */
        }
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
        len(tuser) > 0 && card != nil {
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
