# GO Rules

## Core standards

- Keep all generated code production-ready and strongly typed
- Prefer small focused files and functions over large mixed-responsibility modules.
- Match existing project conventions before introducing new patterns.
- Include meaningful tests for business-critical behavior.
- Never ship placeholder TODO logic in production paths.
- Ensure lint/type/test checks pass before committing.
- Prefer flat patterns over nesting for readability

## Security conventions

- Use explicit naming for modules, services, and handlers.
- Add boundary validation for external inputs and API payloads.
- Keep side effects isolated and observable with logs.
- Favor predictable dependency boundaries and clear ownership.
- Document trade-offs for non-obvious implementation choices.

## App

- App module is a core module, it contains everything related to
our application
- Scope modules per model name e.g. everything related to user goes under user directory
- Provide verbose logs with multiple log levels

