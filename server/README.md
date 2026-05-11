# Portfolio Server

Go API server for the portfolio site.

print('my portfolio')

## Configuration

The server stores portfolio content in PostgreSQL. `DATABASE_URL` is required; the
server does not use `projects.json` or any other JSON file as a data store.

```sh
DATABASE_URL=postgres://portfolio:portfolio@localhost:5433/portfolio?sslmode=disable
```

Required for authenticated content management:

```sh
ADMIN_USERNAME=admin
ADMIN_PASSWORD=change-me
JWT_SECRET=replace-with-a-long-random-secret
```

For the local Angular web admin app, allow browser requests from the Angular
dev server:

```sh
CLIENT_ORIGIN=http://localhost:4200
```

## Docker

Run the API and PostgreSQL together:

```sh
docker compose up --build
```

The API listens on `http://localhost:8080`, and PostgreSQL is exposed on
`localhost:5433` by default to avoid conflicts with a local Postgres instance.
Set `POSTGRES_HOST_PORT=5432` if you want Compose to bind host port `5432`.
Project, certificate, article, work experience, skill, link, tool, music, product, and intro
rows are stored in the `postgres-data` Docker volume. Uploaded local files are
stored in the `uploads` Docker volume.

## Local Run Script

Run PostgreSQL with Docker Compose and the API with Go:

```sh
./scripts/run-server.sh
```

The script starts the Compose `postgres` service, sets local development
defaults, and runs `go run .`. It uses `POSTGRES_HOST_PORT=5433` and
`CLIENT_ORIGIN=http://localhost:4200` by default.

## Sample Data

Seed sample projects, certificates, articles, work experiences, skills, links,
tools, music, products, and intro content into the local PostgreSQL database:

```sh
./scripts/seed-sample-data.sh
```

The seed command replaces only the sample rows by their stable IDs/slugs,
sample article IDs, sample work experience IDs/slugs, sample skill
IDs/categories, sample link IDs, sample tool IDs, sample product IDs/slugs, the
sample music IDs, sample product IDs/slugs, the products section settings row,
and the singleton intro row. It does not wipe
other projects, certificates, articles, work experiences, skills, links, tools,
music, or products. Each seeded collection is capped by the seed command's
sample limit.

## Postman

Import `postman/portfolio-server.postman_collection.json` into Postman. Run the
`Auth / Login` request first; it stores the JWT in the collection variables for
the protected project, certificate, article, skill, link, tool, music, product, and intro
requests.

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

Uploads accept only `.png`, `.jpg`, and `.jpeg` files whose detected content
type matches the extension. The default file size limit is 300 MiB and can be
overridden with `MAX_UPLOAD_BYTES`.

Article cover images are stored as external URLs only; there are no article
image upload routes.

## Endpoints

Read endpoints are public for projects, certificates, links, work experiences,
skill categories, skills, tools, music, daily progress, products, the products section
settings, and intro. Public product list/detail reads return published products only;
authenticated product reads can manage all statuses. All article routes plus
create, update, delete, and upload routes require `Authorization: Bearer <jwt>`.

```txt
POST   /auth/login
GET    /healthz
GET    /links
POST   /links
GET    /links/{id}
PATCH  /links/{id}
PUT    /links/{id}
DELETE /links/{id}
GET    /skill-categories
POST   /skill-categories
GET    /skill-categories/{id-or-slug}
PATCH  /skill-categories/{id-or-slug}
PUT    /skill-categories/{id-or-slug}
DELETE /skill-categories/{id-or-slug}
GET    /skills
POST   /skills
GET    /skills/{id}
PATCH  /skills/{id}
PUT    /skills/{id}
DELETE /skills/{id}
GET    /tools
POST   /tools
GET    /tools/{id}
PATCH  /tools/{id}
PUT    /tools/{id}
DELETE /tools/{id}
GET    /music
POST   /music
GET    /music/{id}
PATCH  /music/{id}
PUT    /music/{id}
DELETE /music/{id}
GET    /daily-progress
POST   /daily-progress
GET    /daily-progress/{id-or-date}
PATCH  /daily-progress/{id-or-date}
PUT    /daily-progress/{id-or-date}
DELETE /daily-progress/{id-or-date}
GET    /products/section
PATCH  /products/section
GET    /products
POST   /products
GET    /products/{id-or-slug}
PATCH  /products/{id-or-slug}
PUT    /products/{id-or-slug}
DELETE /products/{id-or-slug}
GET    /articles
POST   /articles
GET    /articles/{id}
PATCH  /articles/{id}
PUT    /articles/{id}
DELETE /articles/{id}
GET    /work-experiences
POST   /work-experiences
GET    /work-experiences/{id-or-slug}
PATCH  /work-experiences/{id-or-slug}
PUT    /work-experiences/{id-or-slug}
DELETE /work-experiences/{id-or-slug}
GET    /projects
POST   /projects
GET    /projects/{id-or-slug}
PATCH  /projects/{id-or-slug}
PUT    /projects/{id-or-slug}
DELETE /projects/{id-or-slug}
POST   /projects/{id-or-slug}/images/main
POST   /projects/{id-or-slug}/images
DELETE /projects/{id-or-slug}/images/{image-id}
GET    /certificates
POST   /certificates
GET    /certificates/{id-or-slug}
PATCH  /certificates/{id-or-slug}
PUT    /certificates/{id-or-slug}
DELETE /certificates/{id-or-slug}
POST   /certificates/{id-or-slug}/image
DELETE /certificates/{id-or-slug}/image
GET    /intro
PATCH  /intro
PUT    /intro
DELETE /intro
POST   /intro/profile-picture
DELETE /intro/profile-picture
```

Image upload routes expect `multipart/form-data`. Use field `image` for the main image, and `images` or repeated `image` fields for gallery uploads.
