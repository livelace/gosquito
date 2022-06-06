### Credentials:

gosquito is capable to gather secrets from environment variables and [vault](https://www.vaultproject.io/).

### Configuration example (config.toml):

```toml
# ----------------------------------------------------------------------------
# credentials: mattermost default

[cred.mattermost.default]
url      = "https://mattermost.livelace.ru"                                                  # can be string
username = "vault://secret/host/mattermost.livelace.ru/service/mattermost/gosquito,username" # can be vault
password = "vault://secret/host/mattermost.livelace.ru/service/mattermost/gosquito,password" # can be vault
team     = "env://GOSQUITO_MATTERMOST_TEAM"                                                  # can be env

[cred.mattermost.default.vault]
address    = "https://vault.livelace.ru:8200"    # can be string
app_role   = "env://GOSQUITO_VAULT_APP_ROLE"     # can be environment variable
app_secret = "env://GOSQUITO_VAULT_APP_SECRET"   # can be environment variable
```
