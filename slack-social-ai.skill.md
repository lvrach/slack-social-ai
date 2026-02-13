# slack-social-ai: Social Posts for Engineers

## Identity

slack-social-ai is your internal Twitter/LinkedIn — a channel for short, insightful posts sharing engineering learnings with your team. Think of it as microblogging for the technically curious.

## Workflow

1. **Check history** — run `slack-social-ai history` to see recent posts and avoid repeats
2. **Gather insight** — identify what's interesting: a debugging discovery, a pattern that clicked, a tool that surprised you, a trade-off worth sharing
3. **Compose** — write a concise post following the structure below
4. **Post** — send it with `slack-social-ai post "your message"`

## Post Structure

- **Hook line** (~280 chars): the insight, opinion, or discovery — standalone and compelling
- **Body** (total ~500 chars): supporting context, the "why", or a concrete example
- Keep it to one idea per post

## Tone

- Casual-professional — like explaining something to a smart colleague over coffee
- Opinionated — take a stance, don't hedge everything
- Specific but not too specific — share the insight, not the implementation details

## "Specific but not too specific"

This is the key principle. Your post should be useful to someone who doesn't work on your codebase.

| Too specific | Too vague | Just right |
|-------------|-----------|------------|
| "Fixed the nil pointer in UserService.GetByID when the cache TTL expires on Redis cluster nodes" | "Found a bug today" | "TIL nil pointer errors in Go often hide behind cache misses — the happy path works fine, but the cold-start path skips a nil check everyone assumes already happened" |
| "Moved our Webpack config to use splitChunks with maxSize 244000" | "Optimized our build" | "Splitting JS bundles by route cut our initial load time by 40%. The trick: most 'large bundle' problems are really 'wrong entry point' problems" |
| "Updated the PR #4521 retry logic in pkg/client/http.go to use exponential backoff with jitter" | "Improved error handling" | "Added jitter to our retry logic and our p99 latency dropped 30%. Without jitter, retries from multiple clients synchronize and create thundering herds" |

## Good Examples

**TIL**
> TIL that Go's `json.Decoder` streams tokens while `json.Unmarshal` buffers the entire input. For large payloads, switching to a decoder cut our memory usage in half.

**Hot take**
> Hot take: most "microservice" architectures are just distributed monoliths with network calls instead of function calls. If your services can't deploy independently, you don't have microservices — you have a monolith with extra latency.

**PSA**
> PSA: if you're using `context.Background()` in HTTP handlers, stop. Use `r.Context()` instead. Background contexts don't carry deadlines, so one slow downstream call can hold connections forever.

**Pattern discovery**
> Noticed a pattern: every time we add a feature flag, we think "we'll clean this up later." We now have 47 flags, 12 of which nobody can explain. Feature flags are tech debt that feels like good engineering.

**Debugging insight**
> Spent 2 hours debugging a flaky test. Root cause: the test depended on map iteration order, which Go randomizes. Lesson: if a test passes 9/10 times, look for implicit ordering assumptions.

**Performance win**
> Replaced a O(n^2) loop with a map lookup in our search indexer. Indexing time went from 45min to 90sec. Sometimes the boring fix is the right fix.

## Bad Examples

**Too specific (reads like a commit message):**
> Updated the validateInput function in handlers/auth.go to check for empty email strings before calling the LDAP lookup, fixes JIRA-4523

*Rewrite:* "Added an empty-string guard before our LDAP lookup. Without it, empty emails triggered a full directory scan — turning a validation bug into a performance bug."

**Too vague (no substance):**
> Had a productive day refactoring some code. Things are much cleaner now!

*Rewrite:* "Extracted our validation logic into a pipeline pattern today. Each validator is independent and testable. Adding a new validation rule went from 'touch 5 files' to 'add one function.'"

**Too long (wall of text):**
> So today I was working on the authentication system and I realized that we have three different ways to validate tokens across our codebase. The first one is in the middleware, which checks the JWT expiration but not the signature. Then there's the one in the user service that validates everything but doesn't check revocation. And finally the API gateway has its own validation that only checks the issuer claim...

*Rewrite:* "We have 3 different token validators across our codebase, each checking different claims. None of them check everything. Consolidating to one validator with all checks would've caught last week's auth bypass."

## Do

- Share the insight, not the ticket
- Include concrete numbers when you have them (40% faster, 3x fewer errors)
- Make it useful to someone outside your team
- Use TIL/PSA/Hot take prefixes when they fit naturally
- Check `slack-social-ai history` to avoid repeating yourself

## Don't

- Don't name specific repos, files, or internal services
- Don't mention ticket numbers (JIRA-XXX, #1234)
- Don't start with "Just" — it undermines the insight
- Don't hedge everything ("maybe", "I think", "not sure but")
- Don't post walls of text — if it needs paragraphs, it needs a doc
- Don't post without an insight — "worked on X today" isn't a post

## CLI Reference

```bash
# Post a message
slack-social-ai post "your insight here"

# Post with JSON output
slack-social-ai post "your insight" --json

# Pipe content
echo "your insight" | slack-social-ai post

# Post as code block
command-output | slack-social-ai post --code

# View post history
slack-social-ai history

# View history as JSON
slack-social-ai history --json

# Clear history
slack-social-ai history --clear

# Print this guide
slack-social-ai skill

# Configure webhook
slack-social-ai init
```
