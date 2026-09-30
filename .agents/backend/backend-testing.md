---
paths:
  - "server/**/*_test.go"
---

# Backend Testing Standards

## Core Principles
* **Isolation:** Ensure every test block works in isolation with no order dependency or shared global state.
* **Tests for new behavior:** New features and bug fixes ship with unit tests.
* **Coverage Requirement:** Tests must cover "Happy Paths" (core user journeys) and "Edge Cases" (critical error/validation scenarios).

## Naming
* **Naming:** Append test files with `_test` (e.g. `db_test.go`)
