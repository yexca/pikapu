# Reading

## Layout

On wide screens Pikapu shows three panes:

| Pane | Contents |
| --- | --- |
| Sidebar | All articles, Starred, categories, and feeds with unread counts |
| Article list | Articles in the selected view, newest first |
| Reader | The selected article |

Below 1024 px the list and the reader share one column: selecting an article
opens the reader, and **Back** returns to the list. Below 768 px the sidebar
becomes a drawer opened with the sidebar button.

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

The ✓✓ button in the list header marks every unread article in the current
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

Shortcuts are ignored while typing in a field or when a dialog is open.
