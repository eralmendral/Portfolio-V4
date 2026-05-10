# Portfolio Server

Go API server for the portfolio site.

## Configuration

Required for authenticated project management:

```sh
ADMIN_USERNAME=admin
ADMIN_PASSWORD=change-me
JWT_SECRET=replace-with-a-long-random-secret
```

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
