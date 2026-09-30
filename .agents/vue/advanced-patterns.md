---
paths:
  - "**/*.{ts,vue,js}"
---
# Advanced React & TypeScript Patterns

## Logic Locality
* **Local**: Keep the types, interfaces and functions as local as possible
* **Promotion:** if the object is required by 3 or more items you can elevate it

## UI Patterns
* **Composition:** Prefer composition over HOCs.
* **Compound Components:** Use the `Parent.Child` pattern for complex widgets to keep props flat.

## Cache Management
* **Explicit Stale Time:** Define `staleTime` based on data volatility rather than relying on the default (0).
* **Type Guards:** Use User-Defined Type Guards when validating unknown API responses to ensure runtime safety.
