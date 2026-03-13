# Bot Mode Review Collaboration Design

Date: 2026-03-10

## Goal

Define a bot-mode collaboration flow where one Telegram bot executes work and another Telegram bot automatically reviews the result, without requiring the user to manually coordinate relay steps.

## User Model

The user should think in terms of bots, not project names.

Two fixed commands define the workflow:

```text
/set-project <project> @main_bot @review_bot
/task @main_bot <task>
```

Example:

```text
/set-project MikanAnime @main_bot @review_bot
/task @main_bot 修复登录失败问题
```

## Intended Behavior

`/set-project` creates a group-scoped collaboration record:

- active project: `MikanAnime`
- main bot: `@main_bot`
- review bot: `@review_bot`

`/task @main_bot <task>` runs a controlled review workflow:

1. `@main_bot` works on the task in the configured project
2. after the first completion, `@main_bot` relays the result to `@review_bot`
3. `@review_bot` returns review findings, risks, and test suggestions
4. `@main_bot` posts the final summary back to the group

The review loop is single-pass by default to avoid runaway bot-to-bot conversations.

## Why This Model

The existing relay model is project-oriented. That fits `projects` mode, but it is unintuitive in `bot` mode where the user sees named bots in a group chat.

This design moves the user-facing contract to bot identities:

- users bind bots with `@bot`
- project selection is group-scoped
- relay remains an internal mechanism

## Constraints

- Initial scope is Telegram only for `@bot` parsing and bot identity matching
- The first version should support exactly one main bot and one review bot
- Free-form autonomous multi-bot debates are out of scope
- Existing private-chat `/project` and `/project-list` flows remain unchanged

## Data Model Direction

Bot mode needs a new group-scoped collaboration record distinct from the current per-user active-project binding:

- platform
- chat id
- project
- main bot id
- review bot id

Telegram username or mention parsing should resolve to configured bot identities.

## Command Semantics

### `/set-project`

Recommended forms:

```text
/set-project <project> @main_bot @review_bot
/set-project
/set-project remove
```

Behavior:

- set collaboration config for the current group
- show current group collaboration config
- clear current group collaboration config

### `/task`

Recommended form:

```text
/task @main_bot <task>
```

Behavior:

- validate that the mentioned bot is the configured main bot for this group
- route execution to the main bot runtime for the configured project
- automatically trigger one review pass through the configured review bot
- return the final summary through the main bot

## Safety and Control

To keep the system predictable:

- one automatic review pass only
- no recursive review of review responses
- no simultaneous execution by both bots
- the main bot remains the only bot that returns the final answer to the group

## Implementation Direction

Likely implementation steps:

1. add a bot-mode collaboration store for group-level config
2. add Telegram `@bot` resolution against configured bot identities
3. add `/set-project` handling in bot mode
4. add `/task` handling in bot mode
5. wire bot-mode relay using bot identities instead of direct project naming in the user-facing command layer
6. add focused tests for group config, mention parsing, and single-pass review flow

## Open Questions

- Whether `/bind` should remain available in bot mode as a low-level escape hatch
- Whether `@bot` matching should allow both configured bot names and Telegram usernames
- How much of the automatic review transcript should be shown verbatim in group chat
