# trellohub
Microservice for synchronising a Trello-based workflow with GitHub issues.

# Board Setup
Activate GitHub power up by yourself because it needs permissions. You can do away without it anyway.

# Note!
The code is written with least resistance route in mind and doesn't really represent neither good Go practices nor our best effort. We use it internally and only code for what flexibility and error conditions we personally encounter. Use at your own risk.

Code generally not thread-safe, race conditions possible in case of fast event stream.

# Behaviours

- Attachment added to the card in "Repositories List"
  - Checks if URL added is a GitHub URL
  - Creates a label corresponding to the repository
  - Applies the label to the card (multiple labels over one card allowed)
  - Issues from this repository are accepted in the workflow
  - Setup GitHub webhook automatically (NYI)
- Issue created in the repository listed in "Repositories List"
  - Adds a card in "Inbox" at the top
  - Attaches the issue URL to the card
  - Applies the repository label to the card
  - On GitHub assigns the "inbox" label to the issue
  - If issue text contains a checklist, copies the same checklist over Trello
- Card moved between the lists
  - Changes the corresponding label provided the card was moved between lists in service
- Issue labelled on GitHub with a label of the list
  - Moves the card corresponding to the issue to the list corresponding to the label
- User is assigned/unassigned to the issue on GitHub
  - Assigns/unassigns the same user (using a correspondence table) to the card
- @mention is used in description or checklist at Trello or GitHub
  - Replaces the @mention with a corresponding username on the linked resource

# Far Horizon

- Handle renamings and title updates
- Handle forced push of pull request data
- Error reporting
- Uniform logging
- More docu
- Cache (GitHub's request saving technique)
- Block incorrect actions (e.g. trying to move a card over repositories, deleting main attachment etc)
- Extra hook removal
- Overall anti-fragility code
- Sync comments (do we really need it?)
- Pass by reference and stuff
- Treat Trello IDs as large integers maaybe?
- Find a workaround for `issue.String()`
- Reordering items in the checklist
