# Api rules

For web framework we use Gin, and swago for swagger documentation

## Gin Security Rules

### Gin guidance

- Optimize for readability first, then measure before optimization.
- Keep configuration centralized and environment-safe.
- Avoid hidden magic and implicit behavior.

### Security & reliability checklist

- Validate and sanitize all user-controlled inputs.
- Avoid leaking secrets in logs or error responses.
- Fail safely with clear, actionable error messages.
- Add retries/timeouts only where idempotency is guaranteed.

## Swago

- Write API anotation
- Cove all the possible return codes
- Mark protected routes

