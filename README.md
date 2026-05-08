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


## Tech Stack

- **Backend:** Go 1.21, go-imap, go-smtp
- **Frontend:** React, TanStack Router/Query/Virtual

## Install go

Windows
```
.\install-go.ps1
```