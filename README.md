# discord-log

discord-log is a discord chat logger that logs all chat messages, edits and more to a graylog instance.

```shell
go install github.com/jxsl13/discord-log@latest

# or

go install github.com/jxsl13/discord-log@@main
```

## Usage

```shell
discord-log --help
Environment variables:
  DISCORD_TOKEN      bot or personal discord token
  USER_BOT           set to true if the token is a bot token (default: "false")
  GRAYLOG_ADDRESS    udp gelf endpoint

Usage:
  discord-log [flags]

Flags:
  -c, --config string            .env config file path (or via env variable CONFIG)
      --discord-token string     bot or personal discord token
      --graylog-address string   udp gelf endpoint
  -h, --help                     help for discord-log
      --user-bot                 set to true if the token is a bot token
```
