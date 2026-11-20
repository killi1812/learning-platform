# AI Reflection — Lorena

<!--toc:start-->
- [AI Reflection — Lorena](#ai-reflection--lorena)
  - [Where and Why](#where-and-why)
  - [Tools](#tools)
  - [Quality Assurance](#quality-assurance)
  - [Lessons Learned](#lessons-learned)
<!--toc:end-->

## Where and Why

- unit and integration tests like `db_test.go`, `api_test.go`, `dto_test.go`, and setting up in-memory SQLite test cases 
- repetitive code like DTO structs and mapper functions (`dto.go`), Swagger/Swaggo route annotations, and basic Vuetify dialog/table markup (data models and API structure aleady designed)
- unfamiliar parts like configuring `@vue-office` document viewers (`docx`/`pptx`), and debugging specific PostgreSQL migration

- didn't use it for 
  - core domain modeling decisions  
  - deciding how course enrollment and module prerequisites should work
  - structuring the API routes and UI flows were planned out manually first

## Tools

- used Gemini as coding/chat assistant, antigravity
- added markdown project rule files in `.agents/` (`golang-rules.md`, `backend-testing.md`, `css.md`, etc.) so that whenever an assistant generated anything, it followed our project's conventions (like using Vuetify theme variables instead of hardcoded colors and matching our error handling style)

## Quality Assurance
- autoamted tests (`task test`, `task coverage`) along with manual feature testing
- linting and type checking (`task lint`), ran `go vet`, `vue-tsc`, and ESLint/Prettier before committing
- code reviews during merge requests

## Lessons Learned

- Helped with 
  - saved a lot of time on tasks like writing bunch of similar HTTP handler test cases, Swagger comments, and repetitive TypeScript interfaces
  - helped quickly spot edge cases during code review (such as checking MIME types on file uploads and preventing circular module dependencies)

- Wasn't much of a help 
  - Package Structure vs. Go Circular Imports; in the begingnig it suggested to split every domain entity into its own package caused circular dependency errors in Go as soon as we needed bidirectional GORM preloads; solved it by consolidating them into `course`, `content`, and `submission` packages solved it
  - Vuetify & Permission Edge Cases: it sometimes missed rules like using specific themeing or hardcoding values, which were fixed manualy

--- 

>[!note] While useful and capable, I'd keep AI strictly scoped to small, well-defined tasks (like generating test cases for a single function or writing Swagger docs) rather than letting it touch multiple files at once
 