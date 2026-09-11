# Project (insert name here)

<!--toc:start-->
- [Project (insert name here)](#project-insert-name-here)
  - [Seutp](#seutp)
  - [Checking dev env](#checking-dev-env)
  - [Run a project](#run-a-project)
    - [Java version](#java-version)
  - [Brew](#brew)
  - [Links](#links)
<!--toc:end-->


## Seutp

- To setup the dev environment:
  - Install [Taskfile](https://taskfile.dev/docs/installation)
  - Run `task setup`

## Checking dev env

- Run `task check-env`

## Run a project

- Run `task dev`
  - The database will be started in the docker container and stopped after
  - The code will hot-reload so **No** need for reruns
- The project starts on **url:** `http://localhost:8080`

### Java version

- we use Java 25
- check with `java --version` or `task check-java`
- If you have multiple java versions install

### Brew

Brew is a _MacOS_ [package manager](https://en.wikipedia.org/wiki/Package_manager)

To install brew run the following command

```bash
/bin/bash -c "$(curl -fsSL https://raw.githubusercontent.com/Homebrew/install/HEAD/install.sh)"
```

### Docker

- Docker is a container environment where out app will run.
- On _MacOS_ we will use [_colima_](https://colima.run/) as container software

## Links

- [Google Drive for Artefacts](https://drive.google.com/drive/folders/1kL1AuzYxqAfphmBIeG5EyPSEsaUaagDX?usp=drive_link)
- [GitLab](https://git.app.uib.no/fran.cvok/dat250-project#)
