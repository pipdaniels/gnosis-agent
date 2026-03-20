---
trigger: always_on
---

# Project Rules and Guidelines

This `rule.md` outlines the core practices, patterns, and technologies used in this project. All new contributions must adhere to these guidelines to ensure consistency and maintainability.

## 1. Technology Stack
- **Language**: Go (Golang).
- **Web Framework**: Echo (`github.com/labstack/echo/v4`). Use Echo's built-in tools for routing, middleware, and context handling.
- **Templating Engine**: Templ (`github.com/a-h/templ`). HTML templates are written in `.templ` files and compiled into Go code.
- **Database**: MongoDB. Interactions and connections to the database are managed within the `internal/db` package.

## 2. Project Structure
The repository follows a standard Go project layout:
- `configs/`: Configuration files (e.g., YAML files for settings).
- `internal/`: Private application and library code.
  - `agents/, db/, models/, handlers/, services/, web/, etc.`
- `web/`: Frontend assets and templ files.
- `docs/`: Documentation.
- `scripts/`: Shell scripts for building and installing dependencies.
- `tests/`: Integration and related testing code.
- `Makefile`: Defines the build, run, and utility targets for the project.

## 3. Workflow & Tooling

### Makefile
The `Makefile` is the primary entry point for managing the project lifecycle. Do not invoke tools manually if a target exists.
- **Run Locally (Dev)**: `make dev` - Runs the application in development mode with hot-reload (via `air`) and generates templates.
- **Build**: `make build` - Builds the application into the `bin/` directory.
- **Run Locally (Prod)**: `make run` - Starts the compiled application.
- **Docker**: `make docker-build`, `make docker-up`, `make docker-down` for containerization tasks.

### Templ Generation
**CRITICAL**: You must *always* generate templates with `templ generate` before running or building the project.
This is typically handled by running `make generate` or directly within `make dev` and `make build`. 

## 4. Coding Patterns
- **Echo Setup & Middleware**: Handlers and routes are typically registered in `main.go`. We use standard Echo middlewares like `middleware.Logger()`, `middleware.Recover()`, and `middleware.CORS()`.
- **Authentication**:
  - The Web UI routes are protected by a JWT-based authentication mechanism (via cookies).
  - Programmatic API internal endpoints use API Key validation through headers (configured via `middleware.KeyAuthWithConfig`).
- **Configuration Management**: The application fetches config parameters primarily from YAML files handled via the `internal/config` module.
- **Database Management**: The application implements multi-tenant / organization setups gracefully, initializing MongoDB managers (e.g., `db.NewMongoManager`) and closing them on shutdown.
- **Graceful Shutdown**: The core startup logic in `main.go` implements signal handling (`syscall.SIGINT`, `syscall.SIGTERM`) with `context.WithTimeout` ensuring the server cleanly shuts down and database connections are closed correctly.

## 5. Adding New Features
1. **Define/Update Models**: Start by placing your structures in `internal/models/`.
2. **Database Logic**: Place corresponding MongoDB queries / repository structs in `internal/db/`.
3. **Services & Business Logic**: Complex operations and workflows should live in `internal/services/`.
4. **Handlers**: Presenter logic utilizing Echo context should reside in `internal/handlers/`.
5. **Views**: Build UI views by writing `.templ` files in `web/` and remember to run `templ generate`.
6. **Routes**: Register your new handler routes within the protected or public groups in `main.go` as appropriate.
