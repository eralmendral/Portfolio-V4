# App

This project was generated using [Angular CLI](https://github.com/angular/angular-cli) version 21.2.10.

## Development server

To start a local development server, run:

```bash
bun run start
```

Once the server is running, open your browser and navigate to `http://localhost:4200/`. The application will automatically reload whenever you modify any of the source files.

## Code scaffolding

Angular CLI includes powerful code scaffolding tools. To generate a new component, run:

```bash
bun run ng generate component component-name
```

For a complete list of available schematics (such as `components`, `directives`, or `pipes`), run:

```bash
bun run ng generate --help
```

## Building

To build the project run:

```bash
bun run build
```

This will compile your project and store the build artifacts in the `dist/` directory. By default, the production build optimizes your application for performance and speed.

## Hugeicons Pro

The app has `@hugeicons/angular` installed as the Angular icon renderer. Pro icon packages are served from Hugeicons' private registry, configured in `bunfig.toml` to read the token from `HUGEICONS_LICENSE_KEY`.

After adding your license key locally or in CI, install the first Pro style package:

```bash
bun add @hugeicons-pro/core-stroke-rounded
```

Add other `@hugeicons-pro/core-*` style packages the same way when needed. Keep the license key in an environment variable or local `.env` file, not in version control.

## Running unit tests

To execute unit tests with the [Vitest](https://vitest.dev/) test runner, use the following command:

```bash
bun run test
```

## Running end-to-end tests

For end-to-end (e2e) testing, run:

```bash
bun run ng e2e
```

Angular CLI does not come with an end-to-end testing framework by default. You can choose one that suits your needs.

## Additional Resources

For more information on using the Angular CLI, including detailed command references, visit the [Angular CLI Overview and Command Reference](https://angular.dev/tools/cli) page.
