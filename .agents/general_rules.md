## Operational Constraints

- **Stop & Clarify**: If requirements are ambiguous or conflicting, STOP and ask. Never "guess-fill" logic.
- **Assumption Log**: State non-trivial assumptions BEFORE implementation. 
- **The "Why" Test**: Challenge problematic user approaches. Explain the downside (e.g., "This creates an N+1 query") and quantify when possible. Offer one alternative.
- **Scope Lock**: Modify ONLY the requested files/lines. No unsolicited refactoring.
- **Cleanup**: Identify and ask to remove unreachable code/dead imports created by your changes.

## Engineering Standards

- **KISS Principle**: Prioritize standard library solutions over complex abstractions.
- **Consistency**: Match the local variable casing, indentation, and docstring style exactly.
- **Design Patterns**: Use design patterns to keep code clean and readable
- **Test coverage**: Ensure that the test coverage is above 90%
- **Test standard**: Tests need to test actual behaviour don't write hollow tests

## Communication Protocol

- **Pre-Flight**: Provide a brief bulleted plan for multi-step tasks.
- **Post-Flight Summary**: After a task that changed files, end with: changes made, things intentionally skipped, and risks or manual verification steps.
- **Uncertainty**: Use "I am unsure about X" instead of hedging with confident but vague language.
