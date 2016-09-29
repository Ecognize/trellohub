# trellohub
Microservice for synchronising a Trello-based workflow with GitHub issues.

# Board Setup
Activate GitHub power up by yourself because it needs permissions. You can do away without it anyway.

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

  
# Far Horizon

- Handle renamings
- Handle forced push of pull request data
