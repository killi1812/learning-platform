# Learning Platform (DAT250 Project)

<!--toc:start-->
- [Learning Platform (DAT250 Project)](#learning-platform-dat250-project)
  - [Setup](#setup)
  - [Checking dev env](#checking-dev-env)
  - [Run the project](#run-the-project)
    - [Go version](#go-version)
    - [Brew](#brew)
    - [Docker](#docker)
  - [Testing and formatting](#testing-and-formatting)
  - [Documentation](#documentation)
  - [Links](#links)
<!--toc:end-->

## Setup

- To setup the dev environment:
  - Install [Taskfile](https://taskfile.dev/docs/installation)
  - Run `task setup`
    - Copies `example.env` to `.env` if it doesn't exist yet
    - Installs system dependencies via Homebrew (`go`, `docker`, `colima`, `npm`, `node`, `docker-compose`)
    - Installs Go dev tools (`air` for live reload and `swag` for Swagger docs)

## Checking dev env

- Run `task check` (or just `task`) to verify that Go, Node, Docker/Colima, `air`, and `swag` are installed and in your `PATH`

## Run the project

- Run `task dev`
  - Starts PostgreSQL, MongoDB, SeaweedFS, and RabbitMQ in Docker and stops them automatically on exit
  - Generates Swagger documentation
  - Starts both the Go backend (via `air`) and Vue+Vuetify client concurrently with hot-reload
- **URLs:**
  - **Client UI (Vue 3 + Vuetify):** `http://localhost:3000`
  - **Backend API (Go + Gin):** `http://localhost:8080`
  - **Swagger UI:** `http://localhost:8080/swagger/index.html`
  - **RabbitMQ Management:** `http://localhost:15672`
  - **SeaweedFS Filer:** `http://localhost:8888`

### Go version

- We use Go 1.25+
- Check with `go version` or `task check`
- Make sure your Go bin directory (`$(go env GOPATH)/bin` or `~/go/bin`) is in your shell's `PATH` so `air` and `swag` can be found

### Brew

Brew is a _MacOS_ [package manager](https://en.wikipedia.org/wiki/Package_manager)

To install brew run the following command

```bash
/bin/bash -c "$(curl -fsSL https://raw.githubusercontent.com/Homebrew/install/HEAD/install.sh)"
```

### Docker

- Docker is the container environment where our backing services and production app run
- On _MacOS_ we use [_colima_](https://colima.run/) as the container runtime

## Testing and formatting

- Run `task test` to run the Go unit and integration tests
- Run `task coverage` to generate and open the HTML test coverage report
- Run `task lint` to run `go vet` on the backend and ESLint/type-checking on the client
- Run `task format` to format Go code, client code, and regenerate Swagger docs
- Run `task build` to build the production Docker image (which bundles the Vue build into the Go binary)

## Documentation

Project documentation is located in the [`documentation/`](./documentation) folder:

- [Architecture & Technology Stack](./documentation/architecture.md)
- [Implementation, Features & Limitations](./documentation/implementation.md)
- **Team Reflections:**
  - [Lorena](./documentation/lk-reflection.md)
  - [Fran](./documentation/fc-reflection.md)
  - [Kristian](./documentation/kh-reflection.md)
  - [Daniel](./documentation/dn-reflection.md)

## Links

- [Project Description](./projectDescription/README.md)
- [Google Drive for Artefacts](https://drive.google.com/drive/folders/1kL1AuzYxqAfphmBIeG5EyPSEsaUaagDX?usp=drive_link)
- [GitLab](https://git.app.uib.no/hvl-dat-250/26h/semester-project/group-8/learning-platform)
