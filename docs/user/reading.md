# Reading

## Layout

Pikapu has two layouts; choose one in **Settings → Layout**, or switch with
the layout button in the header. The choice is remembered in this browser.

Both layouts share the same controls in the same places:

- The **header** shows the current view and its unread count on the left,
  and on the right when feeds were last updated, **Refresh**, the layout
  switch, the theme menu, and **Settings**.
- The **toolbar** below it holds the search box, the **Unread / All**
  switch, **Mark all as read**, and, in a single-feed view, the feed's **⋯**
  menu. In the hub it lines up with the stream.

### Classic

On wide screens the classic layout shows three panes:

| Pane | Contents |
| --- | --- |
| Sidebar | All articles, Starred, categories, and feeds with unread counts |
| Article list | Articles in the selected view, newest first |
| Reader | The selected article |

Hover an article in the list (on touch screens it is always shown) and use
the ↗ button at its top right to open the original page in a new browser tab
without selecting it. With **Mark articles as read when opened** on, this also
marks the article as read.

Below 1024 px the list and the reader share one column: selecting an article
opens the reader, and **Back** returns to the list. Below 768 px the sidebar
becomes a drawer opened with the sidebar button.

### Hub

The hub works like a message center: one centered stream of cards.

- **For you** (All articles and categories) picks up to ten unread articles
  from the last week. Picks favor recent articles, feeds you often read or
  star, and feeds that rarely publish, and no single feed can take every
  slot. A label says why an article was picked: *You read this often*,
  *Rare update*, or *Just in*. Picks stay in place while you read and are
  chosen again when you refresh or come back to the view.
- **Latest updates** lists everything else by day. Within a day, each feed's
  articles are stacked in one card with a count of new ones; long stacks
  show three articles and a **Show more** button.
- Selecting an article opens the reader in a panel over the stream (full
  screen on phones). Close it with ✕, `Esc`, or Back.

Search, the Unread/All switch, and keyboard shortcuts work the same in both
layouts. While searching, the hub shows only results, without picks.

## Views and Filters

- **All articles** shows every feed. **Starred** shows starred articles.
  Selecting a category shows all of its feeds; selecting a feed shows only
  that feed.
- The **Unread / All** switch above the list applies to every view except
  Starred and is remembered in this browser.
- Articles you read stay in the unread list until you refresh or change
  views, so you can go back to them.

## Search

Type in the search box above the list (or press `/`). Search matches article
titles and content within the current view and ignores the Unread filter.
Press `Esc` to clear it.

## Article Actions

The reader toolbar can star an article, toggle read state, copy its link, and
open the original page. Articles whose feeds contain only a summary show that
summary; use **Read original** to open the full article on the site.

## Mark All as Read

The ✓✓ button in the toolbar marks every unread article in the current
view as read, or only the search results while a search is active. Pikapu
asks for confirmation except in single-feed views.

## Keyboard Shortcuts

| Key | Action |
| --- | --- |
| `J` / `K` | Next / previous article |
| `M` | Toggle read |
| `S` | Toggle star |
| `V` | Open the original page |
| `R` | Refresh the current feed, or all feeds |
| `/` | Focus search |
| `Shift` + `A` | Mark all as read |
| `Esc` | Close the article |
| `Ctrl` + `B` | Show or hide the sidebar |
| `?` | Show all shortcuts |

Shortcuts are ignored while typing in a field or when a dialog is open (the
hub's reader panel is not a dialog for this purpose). In the hub, `J` / `K`
go through the picks first, then the stream.
