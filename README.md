# Whiteboard

A collaborative whiteboard application that enables multiple users to draw and interact in real-time. Built with a React.js and Go.

## Key Features

- Real-time collaborative drawing board
- Multiple tools (shapes, freehand, text) (in development)
- Multi-room support with unique URLs
- User authentication system
- Drawing persistence and history (in development)
- Export as PNG/PDF functionality (in development)


## How to run

### Prerequisites 

- PostgreSQL
- [Nats io](https://nats.io/) server
- [goose](https://github.com/pressly/goose)

### Running backend

- Clone the repo.
- Create app.env. Example:

```
ENVIRONMENT=dev
DB_SOURCE=db_url
PORT=8080
SMTP_HOST=host
SMTP_PORT=port
SMTP_USERNAME=username
SMTP_PASSWORD=pass
NATS_SERVER_URL=<server-url>

```

- Run `make migrate_up` to handle migrations
- Run nats io server
- Run `go run ./cmd/api` for backend app
- Run `go run ./cmd/worker` for email worker


### Running client

- Clone the repo.
- Create `.env.local` file:

```
VITE_BACKEND_URL=http://127.0.0.1:8080/
VITE_WS_URL=ws://127.0.0.1:8080/ws
```

- Install the dependencies with `pnpm i` or `npm i`
- Run `npm run dev -- --open`
