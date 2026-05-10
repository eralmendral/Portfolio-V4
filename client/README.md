# Portfolio Admin Client

LynxJS/Rspeedy web client for the local portfolio API. It manages projects,
certificates, articles, links, and intro content with JWT login, filters, edit
forms, delete actions, and image uploads where the server supports them.

## Setup

```sh
cd client
npm install
cp .env.example .env.local
```

Default API target:

```sh
PUBLIC_API_BASE_URL=http://localhost:8080
```

## Backend

Start the Go API and local PostgreSQL service:

```sh
cd server
./scripts/run-server.sh
```

Seed local sample data:

```sh
cd server
./scripts/seed-sample-data.sh
```

The local server script uses `admin` / `change-me` as the default admin login.

## Client

```sh
cd client
npm run dev
npm run build
```

The dev server runs on `http://localhost:3000`.
