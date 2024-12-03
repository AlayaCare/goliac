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
  - Give Read/Write access to `Content`
- Where can this GitHub App be installed: `Only on this account`
- And Create
- then you must
  - collect the AppID
  - Generate (and collect) a private key (file)
- Go to the left tab "Install App"
  - Click on "Install"
  - On Repository access, select "Only select repositories", and select the team's repository

We will set ip up in Goliac with
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
- Under Repository permissions
  - Give Read/Write access to `Administration`
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
  - On Repository access, select "All repositories" (except if you are doing a test and want to select the repositories to manage)

We will set ip up in Goliac with
- `GOLIAC_GITHUB_APP_ID` environment variable
- `GOLIAC_GITHUB_APP_PRIVATE_KEY_FILE` environment variable


## Networking

### Outbond connections

- By default Goliac use Github APIs (cf `GOLIAC_GITHUB_SERVER` environment variable). If you are using the default Github Cloud (i.e `https://github.com`) and want to firewall the IPs used by Goliac, check https://docs.github.com/en/authentication/keeping-your-account-and-data-secure/about-githubs-ip-addresses for the list of API IPs adresses
- if you enable Slack notification (see [Optional: Slack integration"](./installation.md#optional-slack-integration), it will also try to connect to Slack API IPs
)

### Inbound connections

- Goliac has a public REST API (used by the UI) on the port `18000` by default ( `GOLIAC_SERVER_PORT` environment variable) and on the `localhost` port by default (`GOLIAC_SERVER_HOST` environement variable).
- it has also a second host/port, `GOLIAC_GITHUB_WEBHOOK_HOST` (`localhost` default value) and `GOLIAC_GITHUB_WEBHOOK_PORT` (`18001` default value) if you want to receive webhook events.
- so by default nothing is exposed externally. Except if you are using the Docker image where `GOLIAC_SERVER_HOST` is set to `0.0.0.0` by default.

If you decide to configure Github webhook events (see [Optional: GitHub webhook"](./installation.md#optional-gitHub-webhook) ), it is recommended to setup webhook listener, by setting correctly
- the `GOLIAC_GITHUB_WEBHOOK_HOST` (you need to change the `localhost` default value)
- the `GOLIAC_GITHUB_WEBHOOK_PORT` (`18001` by default)
- the `GOLIAC_GITHUB_WEBHOOK_PATH` (`/webhook` by default)
- and the `GOLIAC_GITHUB_WEBHOOK_SECRET` (empty by default)

### Restricting Goliac UI and REST API

By default the UI (and the REST API) are listening on `localhost` host except in the docker image where it is exposed to `0.0.0.0`. Of course you can change that by setting the `GOLIAC_SERVER_HOST` environment variable.

If you want to open the UI (and the REST APIs) but in a limited way, you will need to use a side-car (in kubernetes) or something similar, to setup a basic authentication, or a better mechanism.

For example a basic authentication using Apache, can be configured like:

```
<IfModule mod_ssl.c>
<VirtualHost *:443>
	ServerName goliac.mydomain.com

        <Location /> #the / has to be there, otherwise Apache startup fails
            #Deny from all
            #Allow from (You may set IP here / to access without password)
            AuthUserFile /etc/apache2/htpasswd/goliac
            AuthName authorization
            AuthType Basic
            #Satisfy Any # (or all, if IPs specified and require IP + pass)
            #            # any means neither ip nor pass
            require valid-user
        </Location>

	ProxyRequests Off
	<Proxy *>
	Order deny,allow
	Allow from all
	</Proxy>
	ProxyPass / http://localhost:8080/
	ProxyPassReverse / http://localhost:8080/

  RewriteEngine on

  ...
</VirtualHost>
</IfModule>
```

## Logs and PII

By default Goliac will logs
- in text format (you can change it via `GOLIAC_LOGRUS_FORMAT` to `json`)
- as info (you can change it via `GOLIAC_LOGRUS_LEVEL` to `warn` or `error`)

Intentionally, with the (default) info level, Goliac will output command it is running, with some PII informations (name of the author of the change, some information on the changes. you can check the `internal/engine/goliac_reconciliator.go` for more details, especially all `logrus.WithFields` code). It is the intented behaviour to be able to collect what Goliac is doing.

It will output something like
```
time="2024-11-10T04:03:14-05:00" level=info msg="teamslug: a_github_team, username: a_username_githubid, role: member" author=author_of_the_commit command=update_team_add_member dryrun=false
```

If you want to restrict this behaviour, you can change the log level (to `warn` or `error`), and you can still keep the audit feature of Goliac, by reviewing the Git history of your teams repository (in Github)