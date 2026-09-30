---
paths:
  - "**/tests/**/*.test.{ts,vue,js}"
---
# Testing, A11y, and Module Imports

## Import Organization
Order imports as follows:
1. **React core:** `react`, `react-dom`.
2. **Third-party:** TanStack, UI libraries, etc.
3. **Local:** Styles, local components, hooks, types.
*Note: Keep related imports grouped; do not mix order.*

## Testing Standards
* **Location:** `tests/` folder within the component's directory.
* **Naming:** Test files must match `ComponentName.test.vue`.
* **Tools:** Use React Testing Library, Jest, and MSW for API mocking.
* **Structure:** `describe` block for component -> `describe` for condition -> `it` for behavior.

## Accessibility (A11y)
* Include descriptive `alt` props on all `<img>` tags.
* Use semantic HTML elements (e.g., `<button>` instead of `<div>` with onClick).
* Ensure proper ARIA roles and keyboard navigation.
* Avoid the `accessKey` attribute.
