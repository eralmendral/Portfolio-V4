# Server Sync Prompt

Use this prompt for client-side work while the Go server and LynxJS client are
being developed side by side.

```text
We are developing the Go server and LynxJS client side by side. Before making any client changes, inspect the current server implementation as the source of truth.

Goal:
Update the client whenever server endpoints, models, auth requirements, request payloads, response shapes, filters, uploads, or docs have changed. Do not assume the previous client API layer is current.

Required workflow:
1. Read the current server routes in `server/main.go`.
2. Inspect the relevant server handlers/models for every exposed resource.
3. Compare those contracts against `client/src/api.ts`, `client/src/types.ts`, and the client UI.
4. Update the client API layer, TypeScript types, and UI navigation/forms to match current server behavior.
5. Preserve existing client behavior for resources that still match the server.
6. Keep the app practical for admin use during development.
7. Run `cd client && npm run build`.
8. Run `cd server && go test ./...`.
9. Report any server/client mismatch that cannot be resolved safely.

Current server facts to verify against the repo:
- `POST /auth/login` returns a JWT.
- Protected writes use `Authorization: Bearer <token>`.
- Projects and certificates now have public `GET` list/detail routes, but create/update/delete/upload routes are protected.
- Articles exist and are currently JWT-protected for all routes.
- Links exist with public `GET` routes and protected write routes.
- Intro exists with public `GET /intro` and protected update/delete/profile-picture routes.
- The client should cover the current admin resources: projects, certificates, articles, links, and intro.

Constraints:
- Do not overwrite unrelated uncommitted server changes.
- Do not commit `.idea/`, `node_modules/`, or build artifacts.
- Keep CORS/local dev support compatible with `CLIENT_ORIGIN=http://localhost:3000`.
- Treat server code as authoritative over stale README/Postman/client code when conflicts exist.
```
