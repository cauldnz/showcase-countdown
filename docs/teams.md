# Your table's stick

The little screen on your table counts down to the showcase. It is also yours
to play with from your coding agent. It can show messages, play tunes, flash
its light, shout to the whole room, and message other tables.

## Connect your agent

The four-digit number in the bottom-left of the screen is your **claim code**.
Everyone at your table uses the same one. The MCP server is at:

```
http://ROUTER:8080/mcp/<your code>
```

Claude Code:

```bash
claude mcp add --transport http stick http://ROUTER:8080/mcp/1234
```

VS Code (`.vscode/mcp.json`):

```json
{ "servers": { "stick": { "type": "http", "url": "http://ROUTER:8080/mcp/1234" } } }
```

Cursor (`.cursor/mcp.json`):

```json
{ "mcpServers": { "stick": { "url": "http://ROUTER:8080/mcp/1234" } } }
```

Then tell your agent to **claim** the stick with your team name.

## What it can do

| Tool | What happens |
| --- | --- |
| `claim` | Names your team. Shows on your stick and to other tables |
| `show` | A message on your screen for a few seconds. `[red]markup[/]` works |
| `play` | A built-in jingle, or a tune you compose: `C5:200 E5:200 G5:400` |
| `jingles` | Lists the built-in jingles |
| `led` | Colour and pattern on the light: solid, blink, breathe |
| `shout` | Every screen in the room shows it, with your name. Once every two minutes |
| `message_team` | A message to one other table's screen and inbox |
| `list_teams` | Who is in the room |
| `inbox` | What other tables sent you |
| `wait_for_button` | Blocks until someone presses a button on your stick |
| `status` | Your stick's state |

## Ideas

- Play `merge` every time a PR merges. A Claude Code hook that calls `play`
  on success is about three lines.
- Show the name of the branch you just pushed.
- Breathe green while tests pass, blink red when they fail.
- Have your agent wait for a button press and then read out your inbox.
- Compose a team anthem in notes. Sharps and flats are fine: `F#4:150 Bb4:150`.

## House rules

- The organiser can lock the room. In the last minute before the showcase
  every command is refused so the finale is not stepped on.
- Loud or looping audio gets a table muted. Be a good neighbour.
- Anyone at any table can see any code. Mischief is allowed; be kind.
