# jira-backup

A backup client to trigger and download backups from your Confluence Wiki space.

## preparation

You need to setup the program within a json file, or you use environment variables.


### environment variables

```
JIRA_BASE_URL="" # Confluence Wiki URL, e.g. https://wiki.example.com
JIRA_SPACE_KEY="" # Wiki Space Key, e.g. MYSPACE
JIRA_TOKEN="" # Wiki Access Token
JIRA_BACKUP_DIR="" # Directory were backup files are stored (optional)
JIRA_S3_BUCKET="" # S3 bucket where the backups are stores (optional)
JIRA_S3_REGION="" # S3 region (optional)
JIRA_S3_KEY_PREFIX="" # S3 key prefix (optional)
JIRA_S3_ACCESS_KEY="" # S3 access key (optional)
JIRA_S3_SECRET_KEY="" # S3 secret key (optional)
JIRA_TIMEOUT=10 # timeout while downloading backup file, e.g. 10 min
JIRA_CONFIG="config.json" # json file from which the variables optional load
```

### config.json

minimal version

```json
{   "baseurl": "https://wiki.example.com",
    "spacekey": "MYSPACE"
}
```

## authentication

For access Confluence Wiki API you need a API token or Personal Access Token. The person needs full space permissions to access space content and permissions to export the space.

Refer to the [Kantega SSO Enterprise](https://kantega-sso.atlassian.net/wiki/spaces/KSE/pages/28180485/API+Tokens#Use-an-API-token) for special implementation of Token auth

## run

start the program on a cronjob or manually as often you needed:

```bash
2025/07/31 19:57:47 ▶️ Triggering backup...
2025/07/31 19:57:48 ⏩ Backup job started: 3256844303
2025/07/31 19:57:48 ⌛ Status  PROCESSING
2025/07/31 19:57:48 🔄 Backup in progress...
2025/07/31 19:57:53 ⌛ Status  FINISHED
2025/07/31 19:57:53 ⌛ Downloading backup file...
2025/07/31 19:57:53 🔄 Uploading to S3...  /tmp/confluence_backup_911634790.zip
2025/07/31 19:57:53 ✅ Backup complete.
```

Only one space will be backup. Configure/copy for more instances.

## ref

see [confluence doc](https://confluence.atlassian.com/doc/back-up-a-space-or-multiple-spaces-1236929929.html)

## credits

Frank Kloeker f.kloeker@telekom.de

Life is for sharing. If you have an issue with the code or want to improve it, feel free to open an issue or an pull request.
