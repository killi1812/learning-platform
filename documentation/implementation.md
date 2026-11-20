# Implementation, Features & Limitations

<!--toc:start-->
- [Implementation, Features & Limitations](#implementation-features--limitations)
  - [Thought and Development Process](#thought-and-development-process)
    - [Workflow and Organization](#workflow-and-organization)
    - [Domain Model Evolution](#domain-model-evolution)
    - [Key Design Decisions and Interpretations](#key-design-decisions-and-interpretations)
  - [Novel Features](#novel-features)
    - [Real-Time Push Notifications via Server-Sent Events (SSE)](#real-time-push-notifications-via-server-sent-events-sse)
    - [In-Browser Multi-Format Document and Slide Viewer](#in-browser-multi-format-document-and-slide-viewer)
  - [Known Limitations, Bugs, and Shortcomings](#known-limitations-bugs-and-shortcomings)
    - [Unfinished Scope](#unfinished-scope)
    - [Technical Limitations](#technical-limitations)
    - [Bugs Encountered and Fixed](#bugs-encountered-and-fixed)
    - [What We Would Change Next Time](#what-we-would-change-next-time)
<!--toc:end-->

## Thought and Development Process

### Workflow and Organization

- used a feature-branch workflow on UiB GitLab (`models`, `login`, `courseAdministration`, `content-pages`, `notifications-and-push`, `submissions`, `quiz-system`, `submissions-rest`, `group-formation`, `Group-submissions`, `grading-individual`, `group-grading`) with Merge Requests into `master`
- all common development commands were standardized in `taskfile.yaml` (`task setup`, `task dev`, `task test`, `task lint`, `task format`, `task build`) so every team member and the GitLab CI pipeline ran the exact same checks
- unit and integration tests alongside backend services and API routes (targeting >90% test coverage) using an in-memory SQLite database and in-memory mocks for SeaweedFS and RabbitMQ

### Domain Model Evolution

- we started from the course UML class diagram and mapped out the entities into Go structs and SQL tables before implementing the API routes
- cohesive domain packages (`course`, `content`, `quiz`, `submission`, `grade`) plus `user`/`auth` and `notification`, which eliminated circular imports while keeping a clean dependency graph (`grade` -> `submission` / `quiz` -> `content` -> `course` -> `user`)
- for `Content` items, we used a two-level Class Table Inheritance pattern where the base `contents` table stores common fields (`id`, `uuid`, `module_id`, `title`, `position`, `visible_from`, `visible_until`, `type`), first-level subtype tables (`files`, `assignments`) reference `content_id` as their primary key, and `Assignment` subtypes (`quizzes`, `coding_exercises`) further share `content_id` while `HANDIN` assignments use the `assignments` row directly
- when implementing individual and group grading, we extracted `Grade` from `submission` into its own `server/grade` package so the grading service could combine manual hand-in grades (`IndividualSubmission` and `GroupSubmission`) with automated quiz scores (`quiz.Attempt`) and course grading scales (`course.GradingScale`) without circular package dependencies

### Key Design Decisions and Interpretations

#### Course Enrollment 

- when a user creates a course, they are automatically enrolled with the `TEACHER` role 
- students can browse the course catalog and self-enroll freely 

#### Course Materials (`StaticPage` -> `File`)

- we initially modeled static pages with an iframe `SourceURL` and `RichText` 
- then expanded and renamed `StaticPage` to `File` backed by SeaweedFS 
- teachers can upload real course files (`.pdf`, `.docx`, `.pptx`, `.md`, `.txt`, image formats) that render directly inside the module view (`FileView.vue`)

#### Module Prerequisites (`DependsOn`)

- modules can depend on a prerequisite module in the same course (validated on both frontend and backend to prevent circular dependency chains) 
- we scoped prerequisite completion to module files
    - when a student opens/views the files in a prerequisite module, a `StudentInteraction` record is saved, unlocking the dependent module

#### Asynchronous Submissions & Two-Phase File Upload

- hand-in assignments support both URL deliverables (e.g., Git repository links) and file uploads (up to 200MB, including archives `.zip`, `.tar.gz`, `.tgz` and documents)
- to ensure students are not penalized for network upload latency close to a deadline, file hand-ins use a two-phase upload flow:
  1. `POST .../submission/file/init` verifies eligibility and issues a short-lived `UploadTicket` that locks the official `SubmittedAt` receipt timestamp
  2. `POST .../submission/file/upload` streams the file to SeaweedFS, consumes the ticket, and publishes a `SubmissionMessage` to RabbitMQ (`202 Accepted`)
- the background `SubmissionConsumer` worker reads from the queue, calculates the attempt number, checks the assignment deadline against the locked `SubmittedAt` timestamp to flag `IsLate`, and saves an `IndividualSubmission` or `GroupSubmission` in PostgreSQL (snapshotting current group members in `group_submission_members` so later group roster changes do not alter past submissions)
- a second background worker (`DeadlineReminderWorker`) checks active assignments every minute and sends idempotent 24-hour and 15-minute deadline reminders (`DeadlineReminder`) to enrolled students who have not yet submitted

#### Interactive Quizzes & Automated Grading

- teachers can create, edit, and publish quizzes inside modules (`QuizEditorView.vue`) with configurable `MaxAttempts`, `PassPercent`, `MaxPoints`, visibility windows, and deadlines
- when a quiz is published (or when a published quiz is updated and attempts are reset), enrolled students are notified automatically (`quiz/notify.go`)
- students take quizzes (`QuizTakeView.vue`) with per-question answer saving (`Attempt` and `Answer`) and automated grading on submit (`quiz/grading.go`): `TRUE_FALSE` questions award 1 point for an exact match, while `MULTIPLE_CHOICE` questions evaluate each option position against the answer key

#### Course Groups

- teachers can create individual groups or bulk-create groups (`POST /api/group/bulk` via `BulkGroupsDialog.vue`) with a configurable `MaxMembers` cap and optional random assignment of currently ungrouped students
- students can browse course groups (`CourseGroupsTab.vue` and `GroupsView.vue`), join an available group, or leave their current group

#### Grading, Configurable Grading Scale & CSV Export

- teachers can grade both individual and group hand-in submissions (`GradesView.vue`, `TeacherSubmissionsDialog.vue`, `GradeDialog.vue`) with numeric scores (validated against `Assignment.MaxPoints`) and written feedback
- grades can be saved as drafts (`IsVisibleToStudent = false`) or released immediately, and teachers can bulk-release all staged grades for an assignment (`POST .../grade/release`), which sends real-time notifications to the student or all snapshotted group members
- each course embeds a customizable `GradingScale` (defaulting to the standard Norwegian university cutoffs: A >= 90%, B >= 80%, C >= 60%, D >= 50%, E >= 40%, F < 40%) used to compute cumulative course percentages and final letter grades across visible hand-in grades and best quiz attempts, with full gradebook CSV export (`GET /api/course/:uuid/grade/export`) protected against OWASP CSV formula injection

## Novel Features

### Real-Time Push Notifications via Server-Sent Events (SSE) - MAIN

- a live notification system (`server/notification/`) that pushes notifications to logged-in users in real time over an SSE stream (`/api/notification/stream`)
- an in-memory `Hub` manages active per-user channels and fans out new `NotificationDto` events without blocking while PostgreSQL remains the source of truth
- integrated across platform workflows: welcome notification on first login, 24-hour and 15-minute assignment deadline reminders, quiz publication and update announcements, submission receipts, grade release notifications, and a dedicated teacher/user view for sending notifications, all surfaced in the top navigation bar's live notification bell

### In-Browser Multi-Format Document and Slide Viewer

- to break LMS feeling like a raw file archive where students have to download every file to read it, we built an embedded viewer (`FileView.vue`)
- it streams course files from SeaweedFS with `Content-Disposition: inline` and renders: 
    - PDFs natively
    - PowerPoint slides via `@vue-office/pptx` 
    - Word documents via `@vue-office/docx` 
    - images
    - Markdown/text cards 
  directly inside the course page, with a fallback download option if rendering fails

## Known Limitations, Bugs, and Shortcomings

### Unfinished Scope

#### Coding Exercises and Extended Question Types

- `CodingExercise` (with starter code repositories and visible/hidden test cases) and additional `QuestionType` enum values (`ORDERING`, `MATCHING`, `FILL_IN_THE_BLANKS`) are modeled in the database schema
but 
- the frontend and automated grading currently implement `HANDIN` assignments and `QUIZ` assignments with `MULTIPLE_CHOICE` and `TRUE_FALSE` questions

#### Academic Year Cycles and Course Lifecycles

- Courses have `Term` (AUTUMN/SPRING) and `Year` (e.g. 2026) attributes used for catalog filtering 
but 
- the platform has 
  - no course lifecycle management (such as marking courses as active or archived)
  - no course rollover functionality to duplicate modules or assignments into a new academic year 
  - no handling of student cohorts progressing across years

### Technical Limitations

#### Single-Instance SSE Hub (`server/notification/hub.go`)

 The notification `Hub` keeps open SSE channels in local process memory (`map[uuid.UUID]map[chan NotificationDto]`). If the backend were scaled horizontally across multiple container replicas, an event published on one instance would not reach SSE streams connected to another instance (which would require Redis Pub/Sub or a RabbitMQ fanout exchange)

#### Simple RabbitMQ Retry Policy

 When the `SubmissionConsumer` encounters a database error, it retries the delivery once after a `200ms` backoff before sending the message to the dead-letter queue (DLQ), rather than using exponential backoff with multiple retry queues

### Bugs Encountered and Fixed

#### PostgreSQL `AutoMigrate` with Non-Null UUIDs
 
 Adding `not null` UUID columns to existing PostgreSQL tables during development caused GORM `AutoMigrate` to fail when rows already existed; we fixed this by generating UUIDs in GORM `BeforeCreate` hooks and adjusting the migration tags

#### Orphaned Files in SeaweedFS on Delete

Deleting a course or module cascaded in PostgreSQL but left the uploaded blobs behind in SeaweedFS. We fixed `DeleteModule` and `DeleteCourse` to collect and purge all associated SeaweedFS storage keys before deleting the DB records

#### Grade Foreign-Key Constraints Across Individual and Group Submissions

Moving `Grade` into `server/grade` and supporting polymorphic grading for both `IndividualSubmission` and `GroupSubmission` caused legacy PostgreSQL foreign-key constraint conflicts (`fk_submissions_grade`, `fk_group_submissions_grade`) on existing databases, which we resolved by using nullable `IndividualSubmissionID` and `GroupSubmissionID` pointers and dropping obsolete constraints during startup migration

### What We Would Change Next Time

#### Structure the Backend by Feature Slices Rather Than Per-Entity Domain Packages

Next time, we would group the project by vertical feature slices (or keep shared GORM database models in a single persistence package while organizing business logic and HTTP handlers by feature slice) from the start to avoid Go package cycles and mid-project refactoring

