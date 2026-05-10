# Portfolio Server

Go API server for the portfolio site.

## Configuration

The server stores project data in PostgreSQL. `DATABASE_URL` is required; the
server does not use `projects.json` or any other JSON file as a data store.

```sh
DATABASE_URL=postgres://portfolio:portfolio@localhost:5433/portfolio?sslmode=disable
```

Required for authenticated project management:

```sh
ADMIN_USERNAME=admin
ADMIN_PASSWORD=change-me
JWT_SECRET=replace-with-a-long-random-secret
```

## Docker

Run the API and PostgreSQL together:

```sh
docker compose up --build
```

The API listens on `http://localhost:8080`, and PostgreSQL is exposed on
`localhost:5433` by default to avoid conflicts with a local Postgres instance.
Set `POSTGRES_HOST_PORT=5432` if you want Compose to bind host port `5432`.
Project rows are stored in the `postgres-data` Docker volume. Uploaded local
files are stored in the `uploads` Docker volume.

## Local Run Script

Run PostgreSQL with Docker Compose and the API with Go:

```sh
./scripts/run-server.sh
```

The script starts the Compose `postgres` service, sets local development
defaults, and runs `go run .`. It uses `POSTGRES_HOST_PORT=5433` by default.

## Sample Data

Seed two sample projects into the local PostgreSQL database:

```sh
./scripts/seed-sample-data.sh
```

The seed command replaces only the two sample rows by their stable IDs/slugs.
It does not wipe other projects.

## Postman

Import `postman/portfolio-server.postman_collection.json` into Postman. Run the
`Auth / Login` request first; it stores the JWT in the collection variables for
the protected project requests.

Local development uploads use server disk by default:

```sh
UPLOAD_STORAGE=local
UPLOAD_DIR=public/uploads/projects
UPLOAD_BASE_URL=/uploads/projects
```

Production uploads should use DigitalOcean Spaces:

```sh
UPLOAD_STORAGE=spaces
DO_SPACES_BUCKET=your-space-name
DO_SPACES_REGION=nyc3
DO_SPACES_KEY=your-access-key
DO_SPACES_SECRET=your-secret-key
DO_SPACES_PUBLIC_BASE_URL=https://your-space-name.nyc3.cdn.digitaloceanspaces.com
DO_SPACES_ACL=public-read
```

Uploads accept only `.png`, `.jpg`, and `.jpeg` files whose detected content type matches the extension. The default file size limit is 300 MiB and can be overridden with `MAX_UPLOAD_BYTES`.

## Endpoints

All `/projects` routes require `Authorization: Bearer <jwt>`.

```txt
POST   /auth/login
GET    /projects
POST   /projects
GET    /projects/{id-or-slug}
PATCH  /projects/{id-or-slug}
PUT    /projects/{id-or-slug}
DELETE /projects/{id-or-slug}
POST   /projects/{id-or-slug}/images/main
POST   /projects/{id-or-slug}/images
DELETE /projects/{id-or-slug}/images/{image-id}
```

Image upload routes expect `multipart/form-data`. Use field `image` for the main image, and `images` or repeated `image` fields for gallery uploads.
