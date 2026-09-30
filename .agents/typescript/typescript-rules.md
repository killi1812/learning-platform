# TypeScript Coding Standards

## Naming & Style
* Use **PascalCase** for types, interfaces, and enums.
* Use **camelCase** for variables, functions, and instances.
* Use **CONSTANT_CASE** for immutable module-level constants.
* Use **single quotes** for JS strings; **double quotes** for JSX attributes.
* No spaces inside curly braces in JSX (e.g., `{prop}`).
* Space before self-closing tags (e.g., `<Component />`).

## Type System
* **No `any`:** Use `unknown` for values of uncertain type.
* **Inference:** Rely on type inference for initialized local variables.
* **Interfaces:** Prefer `interface` for public APIs and props; use `type` for unions/intersections.
* **Nullability:** Prefer `undefined` over `null`.

## Language Features
* Prefer `forEach`/array methods over `for...of` for simple iteration; use `for...of` when the body awaits.
* Use ESM `import/export` syntax exclusively.
