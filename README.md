# Mira Mail

Self-hosted email client with a lightweight Go backend and React frontend.

## Structure

```
mira-mail/
├── website/          # React + Vite frontend
│   ├── src/
│   └── package.json
├── server/           # Go backend
│   ├── *.go
│   └── go.mod
└── mira-mail.exe     # Compiled backend
```

## Quick Start

### 1. Build Frontend

```bash
cd website
npm install
npm run build
```

### 2. Build Backend

```bash
cd server
go mod download
go build -o ../mira-mail.exe .
```

### 3. Run

```bash
./mira-mail.exe
```

The server will:
- Generate an API key (printed on first run)
- Serve the frontend at `http://localhost:8080`
- Expose API at `http://localhost:8080/api`

## API Key

On first run, the server prints your API key:

```
=====================================
  API Key: <64-char hex string>
=====================================
```

Store this in the frontend settings to connect.

## Features

- [x] Single EXE - no dependencies
- [x] Auto-generates API key on first run
- [x] IMAP email fetching
- [ ] SMTP sending (WIP)
- [x] Multiple mailboxes (Inbox, Starred, Sent, Drafts, Trash)
- [x] Full-width email view

## Tech Stack

- **Backend:** Go 1.21, go-imap, go-smtp
- **Frontend:** React, TanStack Router/Query/Virtual, Lucide icons
