# Memos MyGo client

This is a native Memos desktop client. It uses sample notes when no server is
configured. To connect to a server, set this variable before starting the app
with `air`:

- `MEMOS_SERVER_URL`: the server origin, such as `https://memos.example.com`.

The app then asks for a username and password in masked native fields. It sends
them over HTTPS and does not save them. A personal access token may be supplied
as `MEMOS_ACCESS_TOKEN` instead; do not put it in a source file or commit it.
The client uses `/api/memos` and falls back to `/api/v1/memos` for older
servers. It loads all pages. Use **New memo** to create a note or **Edit** to
change the selected note. The editor keeps changes local until you choose
**Save memo**. A successful save reloads the list and opens the saved note. The
sun/moon button at the right of the tab row chooses the System, Light, or
Dark theme; the choice is remembered outside the repository.
