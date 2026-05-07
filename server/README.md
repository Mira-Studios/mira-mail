# Mira Mail Server

Fast, lightweight Go backend for Mira Mail.

## Build

```bash
cd server
go mod download
go build -o mira-mail.exe .
```

## Run

```bash
./mira-mail.exe
```

**First run** - generates API key and prints it:
```
=====================================
  MIRA MAIL SERVER
=====================================
  API Key: abc123...
  Port:    8080
=====================================
```

Config stored in `~/.mira-mail/config.json`

## API

| Endpoint | Method | Description |
|----------|--------|-------------|
| `/api/status` | GET | Server status (no auth) |
| `/api/account` | GET | List accounts |
| `/api/account` | POST | Add account |
| `/api/account/:id` | DELETE | Remove account |
| `/api/summary` | GET | Mailbox counts |
| `/api/mailbox/:name` | GET | List emails |
| `/api/email/:uid` | GET | Get email body |
| `/api/compose` | POST | Send email |

**Auth header:** `Authorization: Bearer <API_KEY>`
