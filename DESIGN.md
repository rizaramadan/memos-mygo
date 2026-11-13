# Memos MyGo client

This app uses MyGo's native Go UI. It does not use HTML or a webview. Without
server settings, the included memos are sample content. With `MEMOS_SERVER_URL`
set, the app reads live memos from the Memos API. It can use an access token
from `MEMOS_ACCESS_TOKEN` or a native username/password sign-in. Credentials
and session tokens stay in memory and must not be saved in this repository.

`tokens.css` is the visual source of truth. The Go theme reads its light and
dark values from that file. Follow shadcn's neutral zinc direction: near-black
primary actions, quiet selected rows, outline icons, subtle pane dividers, and
no blue fills or serif headings. Use Inter for body/UI text and Helvetica Neue
for display headings. Use the compact 14px base, 1.2 type scale (seven sizes),
4px spacing unit, and 8px radius. Use only regular, medium, and semibold weights.
Controls are 44px high on narrow layouts and 36px high on wider layouts. Keep
control edges distinct from the softer decorative divider token. Align sidebar
labels to the left and use tabular numerals for counts and dates.

The desktop home screen has two panes: a tabbed library panel and a memo
reader. The left panel keeps search on top, then two icon tabs (memo list and
library/tags), then one scroll region per tab. Each tab scrolls on its own so
large libraries and long memo lists scale. Selecting a tag returns to the
memos tab with the filter applied. Keep the unified width in `tokens.css`
(`library-width`). At narrow widths, show the memo list first and open a memo
in a full-screen reader with a clear way back. Show real memo tags as filters
in both layouts. Keep lean 28px tabs with underline selection; Edit and Open
options live in the tab row so the note starts right under the divider.
Keep search, sample and live-data states, and username/password sign-in.
A theme menu sits at the right end of the reader tab row (and the mobile
list filter row): System, Light, or Dark with a radio mark on the current
choice. It follows the OS by default and remembers the choice outside the
repository.

The authenticated live client can create and edit Markdown memo content. Keep
edits local until Save, show a pending state, and refresh the memo list after a
successful save. The editor is Obsidian-style Live Preview: one pane shows
rendered blocks (headings, lists, wiki links) and clicking any block edits
just that block inline in a borderless field that matches the rendered
typeface, so there is no box and no layout shift; clicking another block,
switching modes, or saving commits it. Source mode shows the raw Markdown
textarea. Switching modes commits any open block edit and must keep the
unsaved draft. Keep reader text within its measure and prevent horizontal
overflow. Typing an unclosed [[ shows a memo-link completion list docked to
the right window edge on desktop widths (inline below the field on narrow
screens); choosing one inserts the [[title]] (an [[target|alias]] keeps its
alias text).
[[target|alias]] shows only the alias and still opens the target. Markdown
[text](https://…) links render as links in the reader and Live Preview and
open in the browser.
