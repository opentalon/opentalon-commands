# opentalon-commands

Content preparer plugin that parses slash commands and returns `invoke` or `message` for the core. The core runs the built-in **opentalon** plugin to execute actions.

**Repository:** [github.com/opentalon/opentalon-commands](https://github.com/opentalon/opentalon-commands)

## Commands

- `/install skill <url> [ref]` — Install a skill from a GitHub URL
- `/show config` — Show current config (secrets redacted)
- `/commands` or `/help` — List available commands
- `/set prompt <text>` — Set the editable runtime prompt
- `/clear` or `/new` — Clear the current session

## Install from GitHub

OpenTalon can fetch and build the plugin from GitHub. In your config, reference the plugin by `github` and `ref` (no local binary needed):

```yaml
plugins:
  opentalon-commands:
    enabled: true
    insecure: false   # required: allows this preparer to return invoke for opentalon actions
    github: "opentalon/opentalon-commands"
    ref: "main"
    config: {}

orchestrator:
  content_preparers:
    - plugin: opentalon-commands
      action: prepare
      arg_key: text
  # ... other preparers (e.g. hello-world) after this
```

The core will download the repo, build the binary, and pin the resolved commit in `plugins.lock`. The commands preparer should run first so slash commands are handled before the LLM.

## Example config (default with this plugin)

Minimal config that enables opentalon-commands from GitHub and the console channel:

```yaml
models:
  providers:
    deepseek:
      base_url: "https://api.deepseek.com/v1"
      api_key: "${DEEPSEEK_API_KEY}"
      api: openai-completions
      models:
        - id: deepseek-chat
          name: DeepSeek Chat
          input: [text]
          context_window: 128000
          cost: { input: 0.14, output: 0.28 }
  catalog:
    deepseek/deepseek-chat: { alias: deepseek, weight: 100 }

routing:
  primary: deepseek/deepseek-chat

orchestrator:
  rules: []
  content_preparers:
    - plugin: opentalon-commands
      action: prepare
      arg_key: text

channels:
  console:
    enabled: true
    plugin: "./channels/console-channel/console"
    config: {}

plugins:
  opentalon-commands:
    enabled: true
    insecure: false
    github: "opentalon/opentalon-commands"
    ref: "main"
    config: {}

state:
  data_dir: ~/.opentalon

log:
  file: ~/.opentalon/opentalon.log
```

## Build locally

From this directory (when developing inside the opentalon repo):

```bash
go build -o opentalon-commands .
```

Or from the opentalon repo root:

```bash
go build -o plugins/opentalon-commands/opentalon-commands ./plugins/opentalon-commands
```

To use a local binary instead of GitHub, set `plugin: "/path/to/opentalon-commands"` in the plugin entry (and omit `github` / `ref`).

## Standalone repo

To fork or copy this plugin to your own repository:

1. Clone or copy from [github.com/opentalon/opentalon-commands](https://github.com/opentalon/opentalon-commands).
2. In `go.mod`, change the module path if needed (e.g. to `github.com/yourorg/opentalon-commands`) and use a published `github.com/opentalon/opentalon` version; remove the `replace` directive when building outside the opentalon repo.
3. Run `go mod tidy`.
