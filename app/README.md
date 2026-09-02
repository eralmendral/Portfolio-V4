# Portfolio UI

## Personal Laboratory admin workspace

The protected `/admin` route hosts an interactive map of the Personal Laboratory. Its content is configuration-driven:

- `src/app/features/admin/data/personal-laboratory.data.ts` contains every laboratory and the connections between them.
- `src/app/features/admin/models/personal-laboratory.models.ts` defines the extensible data shape.
- `src/app/features/admin/pages/admin-page/` renders the network, interactions, and detail panel.

To add a laboratory, append one `Laboratory` object to `PERSONAL_LABORATORIES`, give it a unique `id`, position it with percentage-based `x` and `y` values, and add any desired link to `LABORATORY_CONNECTIONS`. The node and its complete detail view are generated automatically.

Run `bun start` for development, `bun test` for unit tests, and `bun run build` for a production build.
