---
paths:
  - "**/*.{ts,vue,js}"
---
# State Management & Impact Analysis

## 1. State Colocation
* **Rule:** Keep state as local as possible.
* **Action:** Before reaching for a Global Store (Context), verify if the state can live in a parent component or a URL search param.
* **Lifting State:** Only lift state when two or more sibling components strictly require the same data.

## 2. Side Effect Hygiene
* **Cleanup:** Always provide a cleanup function for subscriptions, timers, or manual DOM observers.
* **Race Conditions:** Use AbortController or a "cancelled" flag for any side effects involving asynchronous logic outside of TanStack Query.

## 3. Impact Analysis (Pre-Implementation)
* **Dependency Check:** Before modifying a shared component in `static/js2/components/`, identify all consumer files.
* **Breaking Changes:** If a prop change breaks existing consumers, suggest a migration plan or use optional props with sensible defaults.
* **Query Cache Ripple:** When modifying a mutation (`useMutation`), identify which Query Keys (`['resource']`) need invalidation to keep the UI in sync.

## 4. Context API Limits
* **Granularity:** Create small, focused Context providers rather than one "God Provider."
* **Optimization:** Wrap Context value objects in `useMemo` to prevent unnecessary re-renders of all consumers.
