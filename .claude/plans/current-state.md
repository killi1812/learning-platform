# Current state: learning-platform (handoff for a later session)

Purpose: describe what exists today so a new session can continue. No code was changed to produce this.

## Symptom
After `task dev`, the UI shows only the "Backend Auth Test" card (see screenshot: connection status ONLINE, username/password fields, "Test Authentication" button). Login succeeds, but the user stays on that screen. There is no home page.

## Root cause
The client is a test harness, not an app.
- `client/src/App.vue` hard-codes `<AuthCard/>` inside `<v-main>`. No `<router-view>`, no conditional rendering.
- `client/src/components/AuthCard.vue` `handleLogin()` (about lines 145-194) POSTs `/api/auth/login`. On success it stores `accessToken` and `refreshToken` in `localStorage`, sets local refs, and shows a green alert plus "Authenticated as: X" and a Reset button. It never navigates or emits.
- No `vue-router`, no `pinia`, no auth store or composable, no pages/views/router/stores directories. `package.json` deps: `vue`, `vuetify`, `@fontsource/roboto`, `@mdi/font`.
- Nothing reads the token on startup, so a refresh loses the "authenticated" state.
- `client/src/components/HelloWorld.vue` is unused Vuetify scaffold.
- The client has no tests (no vitest/jest). The passing "tests" are Go tests in `server/` (for example `server/auth/api_test.go`, `server/info/api_test.go`, `server/web/spa_test.go`).

## Stale UI text in AuthCard
- Title and subtitle say "Spring Boot" and "Vue/Vuetify and Spring Boot"; the backend is Go (Gin, GORM).
- "Default credentials: admin/password or user/password" is wrong. The only seeded user is `superadmin` with the password from `SUPERADMIN_PASSWORD` (`Pa$$w0rd` in `example.env`).

## Backend (Go, all routes under /api)
- `POST /api/auth/login` body `{username, email?, password}`; `username` matches username or email. Returns `{accessToken, refreshToken}` only (no user object). Bad login: 401 with a bare JSON string.
- `POST /api/auth/refresh` body `{refreshToken}` (or token in Authorization header). Unprotected.
- `POST /api/auth/logout` protected, 200 with no body.
- `GET /api/user/my-data` any valid token; current-user endpoint. `UserDto` = `{uuid, username, email, role}`; role is `user` or `admin` (`superadmin` maps to admin).
- Admin only: `POST /api/user`, `GET|PUT|DELETE /api/user/:uuid`, `GET /api/user/all-users`, `GET /api/user/search?query=`. Route order (`/:uuid` before `all-users`/`search`) is worth checking.
- `GET /api/info` public: `{build, version, commitHash, buildTimestamp}`.
- No register endpoint. Users can only be created by an admin via `POST /api/user`.
- JWT: HS256, `Authorization: Bearer <jwt>`. Access token 5 min, refresh token 7 days. User UUID is in `jti`. Refresh tokens live in Mongo sessions; login deletes the previous session (one session per user).
- 401 for missing/invalid token, 403 for wrong role. No CORS code (same-origin via the proxy).
- Seed (`server/util/seed/admin.go`): only `superadmin` (email `superadmin@example.com` or `SUPERADMIN_EMAIL`).
- Swagger at `/swagger/*any` in dev builds only; `server/docs/swagger.json` lists 9 paths.

## Wiring
- Vite dev server port 3000, proxies `/api` to `http://localhost:8080` (`client/vite.config.mts`). Client uses relative `/api/...` with raw `fetch`; no API wrapper.
- `taskfile.yaml`: `setup` (copies `example.env` to `.env`, brew installs, installs air and swag), `check` (default), `dev` (docker compose Postgres + Mongo, air for the backend, `npm run dev` for the client).
- Production: `server/web/spa.go` embeds `web/dist` and falls back to `index.html`; `/api` and `/swagger` paths return JSON 404. `Caddyfile` proxies port 80 to `app:8080`.

## Domain model vs code
`projectDescription/` specifies an LMS (courses, modules, content types, enrollments, grading, groups, messaging). The code implements only users and auth. Messaging and any course domain are not started.

## What a later session would need to do (not done)
1. Add routing (`vue-router`, `/login` and `/` with an auth guard) or conditional rendering in `App.vue`.
2. Add an auth composable that loads the token from `localStorage` and fetches `GET /api/user/my-data`.
3. Make `AuthCard.handleLogin` navigate to home on success; add a `HomeView` with user info, role, and logout.
4. Fix stale labels and credentials text.
5. Decide on token refresh (access token expires in 5 min). The user chose not to handle this for now.
6. Optionally add client tests.

## User decisions this session
- Do not change code; only understand the current state.
- Store plans and notes in this project's `.claude/` folder, not in `~/.claude/plans/`.
- No token-refresh handling for now.
