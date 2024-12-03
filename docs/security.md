# Security

## Security hardening

If you want to narrow down what Goliac is doing, you can create 2 github apps
- one to access only the team's repository
- another one that dont have repository access, but only to the organization administrative APIs

### Teams GitHub App

You need to 
- Register new teams GitHub App
  - in your profile settings, go to `Developer settings`/`GitHub Apps`
  - Click on `New GitHub App`
- Give basic information:
  - GitHub App  name can be `<yourorg>-goliac-app-teams`
  - Homepage URL can be `https://github.com/Alayacare/goliac`
  - Disable the active Webhook
- Under Repository permissions
  - Give Read/Write access to `Administration`
  - Give Read/Write access to `Content`
- Where can this GitHub App be installed: `Only on this account`
- And Create
- then you must
  - collect the AppID
  - Generate (and collect) a private key (file)
- Go to the left tab "Install App"
  - Click on "Install"
  - On Repository access, select "Only select repositories", and select the team's repository

We will expose via
- `GOLIAC_GITHUB_TEAM_APP_ID` environment variable
- `GOLIAC_GITHUB_TEAM_APP_PRIVATE_KEY_FILE` environment variable

### Admin GitHub App

If you already created a Github app (when following the installation instructions), you can use it, but remove the repository access

Else you need to
- Register new GitHub App
  - in your profile settings, go to `Developer settings`/`GitHub Apps`
  - Click on `New GitHub App`
- Give basic information:
  - GitHub App  name can be `<yourorg>-goliac-app` (it will be used in the rulesets later)
  - Homepage URL can be `https://github.com/Alayacare/goliac`
  - Disable the active Webhook
- Under Organization permissions
  - Give Read/Write access to `Administration`
  - Give Read/Write access to `Members`
- Where can this GitHub App be installed: `Only on this account`
- And Create
- then you must
  - collect the AppID
  - Generate (and collect) a private key (file)
- Go to the left tab "Install App"
  - Click on "Install"

We will expose via
- `GOLIAC_GITHUB_APP_ID` environment variable
- `GOLIAC_GITHUB_APP_PRIVATE_KEY_FILE` environment variable


