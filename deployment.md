# DigitalOcean Droplet Deployment Guide

This guide deploys both parts of this repository to a DigitalOcean Droplet:

- `server/`: Go API, PostgreSQL, and upload storage through Docker Compose.
- `app/`: Angular admin frontend built to static files and served by Nginx.

The examples use two hostnames:

- Frontend: `admin.example.com`
- Backend API: `api.example.com`

Replace those with the real domains before running the commands.

## Shared Droplet safety rules

This guide is written for a Droplet that may already host other projects.
Before deploying:

- Do not remove existing Nginx site files or anything in `/var/www` for other
  apps.
- Do not run Docker cleanup commands such as `docker system prune`, `docker
  volume prune`, or `docker compose down -v`.
- Use the Compose project name `portfolio-v4` so this app's containers,
  networks, and volumes do not collide with other Compose projects.
- Use unique localhost-only ports for this app. The examples use API port
  `18080` and Postgres port `15433`.
- Use unique Nginx files: `portfolio-api` and `portfolio-admin`.
- Run `sudo nginx -t` before every Nginx reload.

Check what is already running before choosing ports:

```sh
sudo ss -tulpn
docker ps --format 'table {{.Names}}\t{{.Ports}}\t{{.Image}}'
```

If another service already uses `127.0.0.1:18080` or `127.0.0.1:15433`, choose
different ports and keep the values consistent in the Compose override, Nginx,
health checks, and seed command.

## 1. Prepare DNS

In your DNS provider, create `A` records that point to the Droplet public IP:

```txt
admin.example.com -> YOUR_DROPLET_IP
api.example.com   -> YOUR_DROPLET_IP
```

Wait until both names resolve:

```sh
dig +short admin.example.com
dig +short api.example.com
```

## 2. Log in to the Droplet

```sh
ssh root@YOUR_DROPLET_IP
```

Create an app user if the Droplet does not already have one:

```sh
adduser deploy
usermod -aG sudo deploy
```

Log in as that user for the rest of the deployment:

```sh
ssh deploy@YOUR_DROPLET_IP
```

The deploy script needs limited passwordless `sudo` access for Nginx reloads
and frontend file syncs. Create a scoped sudoers file:

```sh
sudo visudo -f /etc/sudoers.d/portfolio-v4-deploy
```

Add:

```sudoers
deploy ALL=(root) NOPASSWD: /usr/bin/mkdir -p /var/www/portfolio-admin, /usr/bin/rsync, /usr/bin/chown -R www-data\:www-data /var/www/portfolio-admin, /usr/sbin/nginx -t, /usr/bin/systemctl reload nginx
```

Do not grant broad passwordless sudo for the shared Droplet.

## 3. Install system packages

```sh
sudo apt update
sudo apt install -y ca-certificates curl dnsutils git nginx openssl rsync ufw
```

Install Docker:

```sh
curl -fsSL https://get.docker.com | sudo sh
sudo usermod -aG docker "$USER"
```

Log out and back in so the Docker group membership applies, then verify:

```sh
docker version
docker compose version
```

Install Bun for the Angular frontend build. This only changes the `deploy` user
environment and does not replace system-wide runtimes used by other projects:

```sh
curl -fsSL https://bun.sh/install | bash
export BUN_INSTALL="$HOME/.bun"
export PATH="$BUN_INSTALL/bin:$PATH"
bun --version
```

Install Go 1.25.5 if you want to run server tests or the seed command directly
on the Droplet. This installs Go under the `deploy` user home directory instead
of replacing `/usr/local/go`:

```sh
curl -fsSLO https://go.dev/dl/go1.25.5.linux-amd64.tar.gz
mkdir -p "$HOME/.local/go1.25.5"
tar -C "$HOME/.local/go1.25.5" --strip-components=1 -xzf go1.25.5.linux-amd64.tar.gz
echo 'export PATH=$HOME/.local/go1.25.5/bin:$PATH' >> ~/.profile
. ~/.profile
go version
```

## 4. Configure firewall

If UFW is already active, do not reset it. Add only the rules this app needs and
review the existing rules for other projects:

```sh
sudo ufw allow OpenSSH
sudo ufw allow 'Nginx Full'
sudo ufw status
```

If UFW is inactive and every public app on the Droplet is served through Nginx
on ports `80` and `443`, enable it:

```sh
sudo ufw enable
sudo ufw status
```

The backend should only be reached through Nginx. The production Compose
override in the backend setup binds backend and Postgres ports to localhost.

## 5. Clone the repository

```sh
sudo mkdir -p /opt/portfolio-v4
sudo chown "$USER":"$USER" /opt/portfolio-v4
git clone YOUR_REPOSITORY_URL /opt/portfolio-v4
cd /opt/portfolio-v4
```

For later deployments, update this same checkout:

```sh
cd /opt/portfolio-v4
git pull --ff-only
```

## 6. Configure backend environment

Create a production env file for Docker Compose:

```sh
cd /opt/portfolio-v4/server
nano .env.production
```

Use real secret values:

```sh
ADMIN_USERNAME=admin
ADMIN_PASSWORD=replace-with-a-strong-password
JWT_SECRET=replace-with-a-long-random-secret
JWT_ISSUER=portfolio-server
TOKEN_TTL=24h
CLIENT_ORIGIN=https://admin.example.com
API_HOST_PORT=18080
POSTGRES_HOST_PORT=15433
MAX_UPLOAD_BYTES=314572800
```

For local disk uploads on the Droplet, add:

```sh
UPLOAD_STORAGE=local
```

For DigitalOcean Spaces uploads, use this instead:

```sh
UPLOAD_STORAGE=spaces
DO_SPACES_BUCKET=your-space-name
DO_SPACES_REGION=nyc3
DO_SPACES_KEY=your-access-key
DO_SPACES_SECRET=your-secret-key
DO_SPACES_PUBLIC_BASE_URL=https://your-space-name.nyc3.cdn.digitaloceanspaces.com
DO_SPACES_ACL=public-read
```

Generate a strong JWT secret with:

```sh
openssl rand -base64 48
```

Important: `server/compose.yaml` currently uses the internal Postgres user,
password, and database `portfolio`. If you change those values, update both the
`postgres` service environment and the `DATABASE_URL` in `server/compose.yaml`.

Create a production Compose override so backend and database ports are not
published publicly:

```sh
cd /opt/portfolio-v4/server
nano compose.production.yaml
```

```yaml
services:
  app:
    ports:
      - "127.0.0.1:${API_HOST_PORT:-18080}:8080"

  postgres:
    ports:
      - "127.0.0.1:${POSTGRES_HOST_PORT:-15433}:5432"
```

## 7. Start the backend

```sh
cd /opt/portfolio-v4
docker compose -p portfolio-v4 --env-file server/.env.production -f server/compose.yaml -f server/compose.production.yaml up -d --build
```

Check the containers:

```sh
docker compose -p portfolio-v4 -f server/compose.yaml -f server/compose.production.yaml ps
docker compose -p portfolio-v4 -f server/compose.yaml -f server/compose.production.yaml logs -f app
```

Verify the API health endpoint from the Droplet:

```sh
curl -i http://localhost:18080/healthz
```

Expected result: `HTTP/1.1 204 No Content`.

## 8. Build the frontend

Build the Angular app:

```sh
cd /opt/portfolio-v4/app
bun install --frozen-lockfile
bun run build
```

The build output is `app/dist/app/browser`.

Deploy it to Nginx's web directory:

```sh
sudo mkdir -p /var/www/portfolio-admin
sudo rsync -a --delete /opt/portfolio-v4/app/dist/app/browser/ /var/www/portfolio-admin/
sudo chown -R www-data:www-data /var/www/portfolio-admin
```

## 9. Configure Nginx

Confirm the hostnames are not already configured by another site:

```sh
sudo nginx -T | grep -E 'server_name .*api\.example\.com|server_name .*admin\.example\.com' || true
```

If that command returns an existing site for either hostname, stop and decide
whether to reuse that site or choose different hostnames. Do not overwrite an
existing unrelated Nginx file.

Create the API reverse proxy:

```sh
sudo nano /etc/nginx/sites-available/portfolio-api
```

```nginx
server {
    listen 80;
    server_name api.example.com;

    client_max_body_size 300m;

    location / {
        proxy_pass http://127.0.0.1:18080;
        proxy_http_version 1.1;
        proxy_set_header Host $host;
        proxy_set_header X-Real-IP $remote_addr;
        proxy_set_header X-Forwarded-For $proxy_add_x_forwarded_for;
        proxy_set_header X-Forwarded-Proto $scheme;
    }
}
```

Create the frontend site:

```sh
sudo nano /etc/nginx/sites-available/portfolio-admin
```

```nginx
server {
    listen 80;
    server_name admin.example.com;

    root /var/www/portfolio-admin;
    index index.html;

    location = /manifest.webmanifest {
        add_header Cache-Control "no-cache";
        default_type application/manifest+json;
        try_files $uri =404;
    }

    location = /ngsw-worker.js {
        add_header Cache-Control "no-cache";
        try_files $uri =404;
    }

    location = /ngsw.json {
        add_header Cache-Control "no-cache";
        try_files $uri =404;
    }

    location = /index.html {
        add_header Cache-Control "no-cache";
        try_files $uri =404;
    }

    location ~* \.(?:css|js|mjs|png|jpg|jpeg|gif|svg|webp|ico|woff2?)$ {
        add_header Cache-Control "public, max-age=31536000, immutable";
        try_files $uri =404;
    }

    location / {
        try_files $uri $uri/ /index.html;
    }
}
```

Enable both sites and reload Nginx:

```sh
sudo ln -sfn /etc/nginx/sites-available/portfolio-api /etc/nginx/sites-enabled/portfolio-api
sudo ln -sfn /etc/nginx/sites-available/portfolio-admin /etc/nginx/sites-enabled/portfolio-admin
sudo nginx -t
sudo systemctl reload nginx
```

These commands only add the two portfolio site symlinks. They do not disable or
delete existing Nginx sites for other projects.

Verify before TLS:

```sh
curl -i http://api.example.com/healthz
curl -I http://admin.example.com
```

## 10. Enable HTTPS

HTTPS is required for browser installability outside `localhost`. Keep the
frontend on HTTPS before treating the PWA checks as production-ready.

Install Certbot:

```sh
sudo apt install -y certbot python3-certbot-nginx
```

Issue certificates and let Certbot update Nginx:

```sh
sudo certbot --nginx -d api.example.com -d admin.example.com
```

Verify renewal is configured:

```sh
sudo certbot renew --dry-run
```

## 11. Seed data when needed

Only run this if you want sample data in the production database:

```sh
cd /opt/portfolio-v4/server
DATABASE_URL=postgres://portfolio:portfolio@localhost:15433/portfolio?sslmode=disable go run ./cmd/seed
```

The seed command replaces only the known sample records. It does not wipe custom
production records, but avoid running it unless sample content is desired.

## 12. Deploy future changes

Manual deploys and GitHub Actions use the same script:

```sh
cd /opt/portfolio-v4
./scripts/deploy-production.sh all
```

Available scopes are `app`, `server`, `all`, and `auto`.

From the Droplet:

```sh
cd /opt/portfolio-v4
git pull --ff-only
```

Run backend checks and redeploy the backend:

```sh
cd /opt/portfolio-v4/server
go test ./...
cd /opt/portfolio-v4
docker compose -p portfolio-v4 --env-file server/.env.production -f server/compose.yaml -f server/compose.production.yaml up -d --build
```

Rebuild and redeploy the frontend:

```sh
cd /opt/portfolio-v4/app
bun install --frozen-lockfile
bun run build
sudo rsync -a --delete /opt/portfolio-v4/app/dist/app/browser/ /var/www/portfolio-admin/
sudo chown -R www-data:www-data /var/www/portfolio-admin
sudo systemctl reload nginx
```

Check the deployment:

```sh
curl -i https://api.example.com/healthz
curl -I https://admin.example.com
curl -I https://admin.example.com/manifest.webmanifest
curl -I https://admin.example.com/ngsw-worker.js
curl -I https://admin.example.com/ngsw.json
docker compose -p portfolio-v4 -f /opt/portfolio-v4/server/compose.yaml -f /opt/portfolio-v4/server/compose.production.yaml ps
```

After frontend deploys, run Chrome or Edge installability checks against
`https://admin.example.com`: confirm the manifest loads, the service worker is
registered, the app can be installed, and a previously loaded app shell reloads
while offline. API-backed public sections, contact submissions, and admin writes
still require network in this v1 PWA baseline.

## 13. Continuous deployment

DigitalOcean does not detect repository changes by itself. GitHub Actions
detects changes on `main`, waits for the `production` environment approval, SSHs
into the Droplet, and runs `scripts/deploy-production.sh`.

Create a GitHub environment named `production` and add required reviewers if
production deploys should require manual approval.

Add these GitHub Actions secrets:

```txt
DO_HOST=YOUR_DROPLET_IP_OR_HOSTNAME
DO_USER=deploy
DO_SSH_KEY=private SSH key allowed to log in as deploy
FRONTEND_URL=https://admin.example.com
```

Add the matching public key to the Droplet:

```sh
ssh deploy@YOUR_DROPLET_IP
mkdir -p ~/.ssh
nano ~/.ssh/authorized_keys
chmod 700 ~/.ssh
chmod 600 ~/.ssh/authorized_keys
```

The workflow deploys only what changed:

- `app/**`: build frontend, sync `app/dist/app/browser` to `/var/www/portfolio-admin`,
  validate and reload Nginx.
- `server/**`: rebuild and restart only the `portfolio-v4` Compose stack.
- both areas or deployment config: deploy both.

The workflow never runs Docker prune, does not touch other Compose project
names, and does not edit unrelated Nginx site files.

## 14. Roll back

Find the previous commit:

```sh
cd /opt/portfolio-v4
git log --oneline -5
```

Check it out and redeploy:

```sh
git checkout COMMIT_SHA
docker compose -p portfolio-v4 --env-file server/.env.production -f server/compose.yaml -f server/compose.production.yaml up -d --build
cd app
bun run build
sudo rsync -a --delete /opt/portfolio-v4/app/dist/app/browser/ /var/www/portfolio-admin/
sudo systemctl reload nginx
```

Return to the deployment branch later:

```sh
cd /opt/portfolio-v4
git switch main
```

## 15. Useful operations

View backend logs:

```sh
cd /opt/portfolio-v4
docker compose -p portfolio-v4 -f server/compose.yaml -f server/compose.production.yaml logs -f app
```

Restart the backend:

```sh
cd /opt/portfolio-v4
docker compose -p portfolio-v4 -f server/compose.yaml -f server/compose.production.yaml restart app
```

Back up the Postgres database:

```sh
cd /opt/portfolio-v4
docker compose -p portfolio-v4 -f server/compose.yaml -f server/compose.production.yaml exec postgres pg_dump -U portfolio portfolio > portfolio-backup.sql
```

Restore a backup:

```sh
cd /opt/portfolio-v4
docker compose -p portfolio-v4 -f server/compose.yaml -f server/compose.production.yaml exec -T postgres psql -U portfolio portfolio < portfolio-backup.sql
```

Check Nginx logs:

```sh
sudo tail -f /var/log/nginx/access.log /var/log/nginx/error.log
```

## 16. Common problems

If the frontend cannot call the API, confirm all three values match:

```txt
server env:   CLIENT_ORIGIN=https://admin.example.com
browser URL:  https://admin.example.com
API URL:      app configuration points to https://api.example.com
```

If uploads fail through Nginx, confirm `client_max_body_size 300m;` is present
in the API Nginx site and `MAX_UPLOAD_BYTES` is large enough in
`server/.env.production`.

If the API starts locally but fails through the domain, check:

```sh
curl -i http://localhost:18080/healthz
sudo nginx -t
sudo systemctl status nginx
docker compose -p portfolio-v4 -f /opt/portfolio-v4/server/compose.yaml -f /opt/portfolio-v4/server/compose.production.yaml logs app
```
