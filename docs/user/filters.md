# Filters

Filters act on new articles by keyword: they can mark an article as read as
it arrives, or skip it so it is never saved. Use them to quiet sponsored posts
and digests, or to keep only the topics you care about from a busy feed.

## Creating a Filter

- **Settings → Reading → Filters → Manage** lists every filter and has
  **Add filter**.
- The **⋯** menu on a feed has **Add filter…**, which starts a filter for that
  feed.

Each filter has:

| Field | Options |
| --- | --- |
| Applies to | All feeds, or one feed |
| Keywords | One or more, separated by commas or new lines |
| Look in | Title, or Title and content |
| When | Any keyword appears, or No keyword appears |
| Then | Mark as read, or Skip |

**No keyword appears** turns a filter around: "Skip when no keyword appears"
with `Go, Rust` keeps only articles that mention Go or Rust.

When several filters match one article, **Skip** wins over **Mark as read**.

## How Keywords Match

- Letter case is ignored: `sponsored` matches `Sponsored`.
- English and other space-separated words match whole words only. `AI`
  matches "AI tools" and "AI-generated" but not "said"; `sponsor` does not
  match "sponsored", so list both forms if you want both.
- Chinese and Japanese keywords match anywhere in the text: `广告` matches
  "广告位".
- A keyword can be a phrase, such as `weekly digest`.
- **Title and content** searches the article text, not its HTML markup.

## When Filters Run

Filters run while feeds update, on articles Pikapu has not seen before.
Articles that are already stored keep their state, so marking a filtered
article as unread sticks.

When you save a filter, **Also apply to current unread articles** (on by
default for new filters) runs it once over unread articles in its scope:
matches are marked as read or, for **Skip**, removed. Starred articles are
never changed.

## Skipped Articles

A skipped article is not stored, so it does not appear in any view or in
search. If you delete or change the filter, skipped articles that are still
in their feed can appear at the next update, as long as they are newer than
the [retention](settings.md#feed-updates) limit.

Deleting a feed deletes its filters. Filters are not included in OPML exports.
