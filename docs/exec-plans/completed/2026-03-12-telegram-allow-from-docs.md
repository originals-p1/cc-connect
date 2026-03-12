# Telegram Allow-From Docs Execution Plan

## Objective

Make the Telegram setup docs clearly explain how to restrict a bot to specific users with the existing `allow_from` option.

## Affected Modules

- `docs/telegram.md`
- `docs/telegram-usage-guide.md`

## Approach

Update the Telegram guides to surface `allow_from` earlier, add a concrete config example for specific user IDs, and clarify that current restriction is user-ID based while chat-ID whitelisting is still not implemented.

## Validation Strategy

- review the edited Markdown for accuracy and internal consistency
- verify the example config keys match `platform/telegram/telegram.go`

## Rollback Plan

- remove the new `allow_from` guidance from the Telegram docs
- restore the previous wording around optional IDs and future chat-ID whitelisting

## Steps

1. Update the English Telegram setup guide to document `allow_from`.
2. Tighten the Chinese usage guide wording so the user whitelist path is explicit.
3. Review the docs against the Telegram platform implementation.
