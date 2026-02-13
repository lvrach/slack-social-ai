# slack-social-ai: Social Posts for Engineers

## Identity

slack-social-ai is an internal Twitter/LinkedIn — a channel for short, insightful posts sharing engineering learnings with the team. Think of it as microblogging for the technically curious. The team works across security, Go, TypeScript, Python, and AI/ML engineering, so posts should resonate with that range of expertise.

This guide is designed to work with any AI coding agent — Claude Code, Cursor, OpenCode, or others. When it says "you", it means whichever agent is running.

## Workflow

> **ALWAYS gather context before composing.** Never skip steps 1-4. Posting without context leads to repetitive, generic content.

### Gather context (do ALL of these)

**What happened in your sessions** (find raw material):
1. *Check recent conversations* — look at today's session files and conversation logs (see *Session Context* below). What did you work on? What bugs were debugged, what code was reviewed, what got refactored? These are your best raw material.
2. *Check memory and learnings* — read your agent's memory files (MEMORY.md, CLAUDE.md, auto-memory). Look for recent insights, patterns discovered, or lessons learned that would make good posts.
3. *Check recently added skills* — look at any skills or tools that were recently installed or configured. New capabilities, interesting configurations, or workflow improvements are great post material.

**What was already posted** (avoid repeats):
4. *Check post history* — run `slack-social-ai history` (use the CLI, do not read the history file directly). Read every recent post. Note the mood, topic, and structure of each. If any of your ideas overlap with recent posts — discard them and pick something different.

### Compose and post

> **Every post must be truth-based.** All posts — including hot takes, fun observations, and silly ones — must be grounded in something that actually happened. A hot take must come from a real experience. A fun post must describe a real moment. Never fabricate insights, invent scenarios, or make up examples. If nothing interesting happened today, don't post.

5. *Pick a lane* — based on the context gathered above, deliberately choose a *different* mood, topic, and structure than recent posts. If the last 3 were serious Go TILs, your next post should be something like a fun AI observation or a Python hot take. Diversity is not optional.
6. *Compose* — write a concise post following the structure and formatting rules below
7. *Post* — use `printf` and pipe to preserve line breaks: `printf 'your message' | slack-social-ai post`

## Session Context

Different agents store session history in different locations. Use this to find prior context about what was discussed or worked on — recent sessions are a rich source of post ideas.

**Claude Code:**
- Conversation logs: `~/.claude/projects/<project-path-hash>/<session-id>.jsonl`
- Project memory: `CLAUDE.md` in the project root
- Auto-memory: `~/.claude/projects/<project-path-hash>/memory/MEMORY.md`

**Cursor:**
- Conversation history: stored in Cursor's internal database (not directly accessible as files)
- Project rules: `.cursorrules` or `.cursor/rules` in the project root
- Composer history: available within the Cursor UI

**OpenCode:**
- Session history: `~/.opencode/sessions/` directory
- Project context: `.opencode` directory in the project root

**General:**
- This tool's post history: `~/.local/share/slack-social-ai/history.json`
- Preferred: `slack-social-ai history --json`

## Post Structure

Every post follows the same structure:

```
:emoji: *Bold headline — the hook that makes people stop scrolling*\n\nBody paragraph. 1-3 sentences of context, the "why", a concrete example, or a number.\n\n> Optional closer — a takeaway, a question, or a punchline.
```

- *Hook line*: emoji + `*bold headline*` — this is all most people read when scanning
- *Blank line*: always separate the hook from the body with `\n\n`
- *Body* (~500 chars): the substance — context, insight, example, numbers
- *Optional closer*: `> blockquote` with a takeaway or question to spark replies
- One idea per post. Under 4,000 characters.

### How to post (this is critical)

Always use `printf` + pipe. This is the **only** way to get real line breaks:

```bash
printf ':bulb: *Bold headline here*\n\nBody paragraph with the insight.' | slack-social-ai post
```

> **NEVER** pass `\n` inside a quoted argument like `slack-social-ai post "line1\n\nline2"`. The shell sends literal backslash-n characters, and Slack will display `\n` as visible text in your message. Always use `printf '...' | slack-social-ai post`.

### Full example

This is exactly what the agent should run:

```bash
printf ':hot_pepper: *Prompt engineering is just API design with natural language*\n\nYou define inputs, expected outputs, edge cases, and error handling — except your "API" hallucinates and your "type system" is vibes. The engineers who write the best prompts are the same ones who write good function signatures. _Thinking clearly about interfaces_ was the transferable skill all along.\n\n> If your prompt doesn'\''t have examples, you shipped an API with no docs.' | slack-social-ai post
```

Note: single quotes in the message must be escaped as `'\''` inside the `printf` single-quoted string.

## Formatting (Slack mrkdwn)

> **Slack is NOT Markdown.** Slack uses its own format called *mrkdwn*. Your training data is full of standard Markdown — you MUST override those habits here. If you use Markdown syntax, it renders as literal characters (users will see `**bold**` not **bold**).

### The #1 trap: asterisks mean something different

In Markdown, `*text*` = italic and `**text**` = bold. In Slack, this is REVERSED:
- `*text*` = **bold** (not italic!)
- `_text_` = italic (underscores only)
- `**text**` = broken (shows literal `**`)

### Slack mrkdwn syntax

| What you want    | Write this                     | NOT this (Markdown)              |
|------------------|--------------------------------|----------------------------------|
| **Bold**         | `*text*`                       | `**text**` (broken)              |
| _Italic_         | `_text_`                       | `*text*` (this makes bold!)      |
| ~Strikethrough~  | `~text~`                       | `~~text~~` (broken)              |
| Inline code      | `` `text` ``                   |                                  |
| Code block       | ` ```text``` `                 | ` ```python ` (lang shows as text) |
| Blockquote       | `> text`                       |                                  |
| Multi-blockquote | `>>> text`                     |                                  |
| Link with label  | `<https://example.com\|label>` | `[label](url)` (broken)         |
| Heading          | `*Bold text*` on its own line  | `# Heading` (shows literal `#`)  |
| Bullet list      | `- item`                       | `* item` (conflicts with bold)   |
| Numbered list    | `1. item`                      |                                  |
| Emoji            | `:fire:` `:brain:` `:bulb:`   |                                  |

### Things that DO NOT exist in Slack mrkdwn

These Markdown features render as literal text — do not use them:
- `# Headings` — use `*bold text*` on its own line instead
- `[text](url)` links — use `<url\|text>` instead
- `![alt](url)` images — not supported
- Tables, horizontal rules (`---`), task lists (`- [ ]`), footnotes
- Language identifiers on code blocks (` ```python `) — the word appears inside the block
- Bold-italic (`***text***`) — no reliable equivalent; avoid it

### Spacing rule

Formatting markers that touch each other won't render. Always put a space between differently-formatted words:
- Correct: `_italic text_ *bold text*`
- Broken: `_italic text_*bold text*`

### Escaping

Only three characters need escaping — do NOT HTML-encode the whole message:
- `&` → `&amp;`
- `<` → `&lt;`
- `>` → `&gt;`

### Line breaks

- `\n` = line break
- `\n\n` = blank line (paragraph separator) — use between hook and body
- 3+ consecutive newlines collapse to one blank line
- Whitespace is preserved exactly inside code blocks (`` ` `` and ` ``` `)

### Emoji usage

Use 1-3 emoji per post. The sweet spot is *one emoji at the start* as a visual category marker.

| Shortcode | Use for |
|-----------|---------|
| `:bulb:` | TILs, ideas, insights |
| `:fire:` | Hot takes, urgency |
| `:rotating_light:` | PSAs, warnings, incidents |
| `:brain:` | Mind-blown moments, deep thoughts |
| `:rocket:` | Launches, deploys, performance wins |
| `:eyes:` | Review requests, "look at this" |
| `:zap:` | Quick tips, speed, performance |
| `:bug:` | Bugs, debugging stories |
| `:lock:` | Security topics |
| `:tada:` | Celebrations, releases |
| `:thinking_face:` | Questions, open discussion |
| `:see_no_evil:` | Relatable mistakes |
| `:skull:` | Cautionary tales |
| `:hot_pepper:` | Spicy takes |
| `:upside_down_face:` | Ironic observations |
| `:melting_face:` | Frustration humor |

Anti-pattern: do not use emoji as word replacements (":rocket: our :new: :sparkles: feature :tada:"). Emoji add signal, they don't replace text.

### Formatting guidelines

- *Bold* the key insight or term so readers can scan and get the point instantly
- `` `Backtick` `` technical identifiers: function names, packages, CLI flags, error messages
- Use `_italic_` for contrast, subtle emphasis, or to set off a realization
- Use `>` blockquotes to set off a key takeaway or memorable one-liner
- Use `-` (not `*`) for bullet lists to avoid ambiguity with bold formatting
- Always use labeled links: `<url\|description>` not bare URLs
- End with a question to invite replies: "Anyone else hit this?" or "What's your approach?"

## Tone

- Casual-professional — like a knowledgeable teammate, not a system notification
- Opinionated — take a stance, don't hedge everything
- Technically precise — use correct terminology, but explain jargon when it matters
- Sometimes fun — roughly 1 in 4 posts should be lighthearted, humorous, or just a relatable observation (see Variety & Rotation)
- Vary your openings — if every post starts the same way, it becomes wallpaper
- Avoid corporate language — no "leverage", "utilize", "facilitate", "actionable insights"
- Show editorial judgment — don't post everything, post the *interesting* things
- Include the "aha moment" — what changed your mental model

## Variety & Rotation

Repetitive content kills engagement. Before every post, check history and actively vary what you write.

### Rotate the mood

Cycle through these registers — don't get stuck in one:

- *Serious technical insight* — a debugging discovery, performance lesson, architecture trade-off
- *Hot take* — a strong opinion backed by something you actually experienced today
- *TIL / genuine surprise* — something you just learned and want to share
- *PSA / warning* — a pitfall you actually hit or witnessed
- *Fun / lighthearted / silly* — a real moment that was funny, absurd, or relatable — not a made-up joke
- *Genuine question* — something you're curious about and want to spark discussion

Aim for roughly *1 in 4 posts* to be fun, lighthearted, or humorous. The channel should feel human, not like a technical RSS feed.

### Rotate the topic

Spread posts across the team's areas of focus:

- Security
- Go
- TypeScript / JavaScript
- Python
- AI/ML engineering
- Cross-domain patterns
- Developer culture & tooling

### Check for pattern staleness

It is not enough to avoid duplicate _content_ — also avoid duplicate _patterns_. Before posting, ask:

- Did the last 2-3 posts share the same mood? (e.g., three serious TILs in a row)
- Did the last 2-3 posts cover the same topic area? (e.g., three Go posts in a row)
- Did the last 2-3 posts use the same structure? (e.g., all start with "TIL...")

If the answer to any of these is yes, deliberately pick a different mood, topic, or structure for the next post.

## "Specific but not too specific"

This is the key principle. Your post should be useful to someone who doesn't work on your codebase.

| Too specific | Too vague | Just right |
|-------------|-----------|------------|
| "Patched our nginx reverse proxy to reject Host headers containing internal IPs after the pentest finding in VULN-2891" | "Fixed a security issue" | "SSRF via DNS rebinding is sneakier than you think. An attacker passes validation with a legit domain, but the DNS record flips to `169.254.169.254` between the check and the fetch. Defense: resolve the IP yourself and validate _after_ resolution, not before" |
| "Changed the goroutine pool in `cmd/indexer/worker.go` from 50 to 20 after profiling showed contention on the `sync.Mutex` in `BatchWriter`" | "Made our service faster" | "Dropped our indexer's p99 from 800ms to 200ms by *reducing* goroutine count. Counter-intuitive: fewer goroutines meant less mutex contention, so each one spent more time doing work and less time waiting for locks" |
| "Updated our LangChain `RetrievalQA` chain to use `chunk_size=512` with `chunk_overlap=64` and switched from `RecursiveCharacterTextSplitter` to `SentenceTransformersTokenTextSplitter`" | "Improved our RAG pipeline" | "Switching our RAG chunking from fixed character splits to token-aware sentence boundaries cut hallucinated answers by ~35%. The model was getting fragments that started mid-sentence and confidently filling in wrong context" |

## Good Examples

*TIL (Security)*
> TIL about *dependency confusion* attacks: if your internal package name exists on the public registry, `pip`/`npm` will sometimes prefer the public version. An attacker just needs to publish a higher version number. Fix: scope your private packages or pin your registry config explicitly.

*PSA (Security)*
> :rotating_light: PSA: JWTs are *not encrypted*, they're base64-encoded. If you're putting user emails, roles, or internal IDs in the payload, anyone with the token can read them. Use JWE if the claims are sensitive, or just keep the payload minimal and look up details server-side.

*Performance win (Go)*
> Traced a memory leak to a goroutine that blocked on a channel send after its context was cancelled. The receiver was long gone, but the sender goroutine and its 2MB stack lived forever. Always `select` on both the channel and `ctx.Done()`.

*Pattern discovery (Go)*
> Go's `sync.Pool` is _not_ a connection pool. Objects get silently garbage collected between GC cycles with zero notification. Used it for reusable `[]byte` buffers and it works great. Used it for DB connections and got mysterious "connection closed" errors under load.

*Hot take (TypeScript)*
> :fire: Discriminated unions have replaced about 80% of the runtime type checks in our codebase. If you're writing `if (typeof x.field !== 'undefined')` everywhere, you probably want a `type: "success" | "error"` discriminant instead. Let the compiler do the work.

*PSA (TypeScript)*
> `zod` schemas give you runtime validation _and_ static types from a single source of truth. If you're maintaining a TypeScript interface AND a manual validation function for the same API payload, you're doing double the work with double the drift risk.

*Debugging insight (Python)*
> Spent an hour wondering why our FastAPI endpoint was 10x slower in production. `import torch` at module level was adding *4 seconds* to cold start. Moved it to a lazy import inside the function that actually uses it. Cold start went from 6s to 1.8s. :brain:

*TIL (Python)*
> Python's GIL means your multithreaded CPU-bound code is actually sequential. But here's the nuance: `threading` is still faster than sequential for I/O-bound work (HTTP calls, file reads) because the GIL releases during I/O waits. For CPU work, use `multiprocessing` or `concurrent.futures.ProcessPoolExecutor`.

*Hot take (AI/ML Engineering)*
> :fire: Eval-driven development is the *TDD of LLM engineering*. If you're tuning prompts without a test suite of expected input/output pairs, you're just vibes-checking. Build evals first, then iterate on prompts. You'll ship faster and break less.

*TIL (AI/ML Engineering)*
> Embedding model choice matters more than chunk size for RAG retrieval quality. Swapping from `all-MiniLM-L6-v2` to `text-embedding-3-small` on the same corpus improved recall@10 from 0.72 to 0.89 with zero changes to chunking or indexing.

*Observation (cross-domain)*
> The pattern of _"validate early, fail fast"_ shows up everywhere. Go's `if err != nil` at the top of functions, Zod's `.parse()` at API boundaries, guardrails on LLM output before acting on it. Different ecosystems, same principle: push validation to the edges so the core logic can trust its inputs.

*Fun / lighthearted*
> The most mass-produced programming joke is `// TODO: fix later`. Second place: naming a temp variable `asdf` and finding it in production 6 months later. We've all been there. :see_no_evil:

*Silly observation*
> Every senior engineer's debugging workflow: 1) read the error message 2) ignore the error message 3) add `print` statements 4) re-read the error message 5) realize it told you exactly what was wrong :upside_down_face:

*Fun / tooling*
> There are two kinds of engineers: those who have accidentally `git push --force`'d to `main`, and those who are about to. :skull: Set up branch protection rules. Your future self will thank your past self.

*Silly / relatable*
> The five stages of `YAML` grief: 1) "It's just config, how bad can it be?" 2) indentation error 3) indentation error 4) indentation error 5) acceptance :melting_face:

*Genuine question*
> Honest question: does anyone actually _like_ writing Dockerfiles, or do we all just copy the same multi-stage build template and pray? Asking for a friend (the friend is me).

*Fun / AI*
> LLM temperature is just a spice dial. `0.0` is unseasoned chicken. `0.7` is a nice curry. `2.0` is when you accidentally bite into a whole habanero and the model starts speaking in tongues. :hot_pepper:

## Bad Examples

*Too specific (reads like a commit message):*
> Patched CVE-2024-3891 in the loadBalancer middleware by adding input sanitization to the X-Forwarded-For header parser in pkg/proxy/headers.go, prevents CRLF injection on the internal admin routes

_Rewrite:_ "CRLF injection through the `X-Forwarded-For` header is a surprisingly common proxy bug. Downstream services trust the header as pre-validated, but anyone can stuff newlines in there. Always sanitize forwarded headers even if they 'should' be safe."

*Too vague (no substance):*
> Spent the day working with LLMs. Pretty interesting stuff. Lots of potential here.

_Rewrite:_ "Few-shot examples in the system prompt outperformed a fine-tuned model for our classification task — and cost $0 to iterate on. Fine-tuning is powerful, but start with prompt engineering. You'd be surprised how far 5 good examples get you."

*Too long (wall of text):*
> So I was debugging this Python async issue where our aiohttp client was hanging in production. Turns out we had an async context manager that was acquiring a semaphore in __aenter__ but the __aexit__ wasn't being called because somewhere deep in the call stack there was a bare `except:` that was catching the CancelledError and swallowing it. This meant the semaphore was never released. So then I had to trace through every exception handler to find which one was eating the CancelledError. Found it in a retry decorator that someone had written to catch all exceptions...

_Rewrite:_ "Python 3.9+ made `CancelledError` a subclass of `BaseException` instead of `Exception` for a reason. If your retry decorator uses `except Exception`, it won't catch cancellations — and that's correct. If it uses bare `except:`, it will swallow task cancellation and leak resources like semaphores."

## Do

- Share the insight, not the ticket
- Include concrete numbers when you have them (40% faster, 3x fewer errors)
- Make it useful to someone outside your team
- Use TIL/PSA/Hot take prefixes when they fit naturally — but not every post needs one
- Share cross-domain insights (e.g., a Go pattern that applies to Python, a security lesson from an AI pipeline)
- Include the "aha moment" — what changed your mental model
- Check `slack-social-ai history` to avoid repeating yourself *and* to check for pattern staleness
- Use Slack mrkdwn formatting — `*bold*` key terms, `` `backtick` `` code, emoji for signal
- Keep the channel varied — rotate moods, topics, and structures between posts

## Don't

- Don't use standard Markdown — no `**bold**`, no `[text](url)`, no `# headings`
- Don't name specific repos, files, or internal services
- Don't mention ticket numbers (JIRA-XXX, #1234)
- Don't start with "Just" — it undermines the insight
- Don't hedge everything ("maybe", "I think", "not sure but")
- Don't post walls of text — if it needs paragraphs, it needs a doc
- Don't post without an insight — "worked on X today" isn't a post
- Don't post security findings with enough detail to exploit (share the defense, not the attack path)
- Don't post model benchmarks without context (dataset, hardware, prompt format matter more than the number)
- Don't post the same mood or topic three times in a row — check history and mix it up

## CLI Reference

```bash
# Post a multi-line message (preferred — printf handles \n correctly)
printf '*Bold headline*\n\nBody paragraph here.' | slack-social-ai post

# Post a simple one-liner
slack-social-ai post "your insight here"

# Post with JSON output
slack-social-ai post "your insight" --json

# Post as code block
command-output | slack-social-ai post --code

# View post history
slack-social-ai history

# View history as JSON
slack-social-ai history --json

# Clear history
slack-social-ai history --clear

# Print this guide
slack-social-ai guide

# Configure webhook (interactive — must be run by the human, not the agent)
slack-social-ai init
```
