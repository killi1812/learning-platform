## Language

- **ALWAYS** Use Standard technical English

## Tools

### codebase-memory-mcp

- When navigating the codebase always use the codebase-memory-mcp MCP server over reading files directly
- Read files only when you need to find out what logic is behind the code

### OpenWolf

- Check .wolf/anatomy.md before reading any project file
- Check .wolf/cerebrum.md Do-Not-Repeat list before generating code
- After writing or editing files, update .wolf/anatomy.md and append to .wolf/memory.md
- After receiving a user correction, update .wolf/cerebrum.md immediately (Preferences, Learnings, or Do-Not-Repeat)
- LEARN from every interaction: if you discover a convention, user preference, or project pattern, add it to .wolf/cerebrum.md. Low threshold — when in doubt, log it.
- BEFORE fixing any bug or error: read .wolf/buglog.json for known fixes
- AFTER fixing any bug, error, failed test, failed build, or user-reported problem: ALWAYS log to .wolf/buglog.json with error_message, root_cause, fix, and tags
- If you edit a file more than twice in a session, that likely indicates a bug — log it to .wolf/buglog.json
- When the user asks to check/evaluate UI design: run `openwolf designqc` to capture screenshots, then read them from .wolf/designqc-captures/
- When the user asks to change/pick/migrate UI framework: read .wolf/reframe-frameworks.md, ask decision questions, recommend a framework, then execute with the framework's prompt

## AI generation behavior

- Generate minimal diffs rather than full rewrites.
- Explain risky changes before applying them.

## Operational Constraints

- **Stop & Clarify**: If requirements are ambiguous or conflicting, STOP and ask.
Never "guess-fill" logic.
- **Assumption Log**: State non-trivial assumptions BEFORE implementation.
- **The "Why" Test**: Challenge problematic user approaches. Explain the downside
(e.g., "This creates an N+1 query") and quantify when possible. Offer one alternative.
- **Scope Lock**: Modify ONLY the requested files/lines. No unsolicited refactoring.
- **Cleanup**: Identify and ask to remove unreachable code/dead imports created by
your changes.

## Engineering Standards

- **KISS Principle**: Prioritize standard library solutions over complex abstractions.
- **Consistency**: Match the local variable casing, indentation, and docstring
style exactly.
- **Design Patterns**: Use design patterns to keep code clean and readable
- **Test coverage**: Ensure that the test coverage is above 90%
- **Test standard**: Tests need to test actual behaviour don't write hollow tests
- **Use Interfaces**: First define an interface and then implement the change, don't
redefine interface if it si not needed

## Communication Protocol

- **Pre-Flight**: Provide a brief bulleted plan for multi-step tasks.
- **Post-Flight Summary**: After a task that changed files, end with: changes made, things intentionally skipped, and risks or manual verification steps.
- **Uncertainty**: Use "I am unsure about X" instead of hedging with confident but vague language.
