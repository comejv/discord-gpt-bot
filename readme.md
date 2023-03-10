# Go Chat GPT Discord bot [![Go](https://github.com/comejv/discord-gpt-bot/actions/workflows/go.yml/badge.svg)](https://github.com/comejv/discord-gpt-bot/actions/workflows/go.yml)

Discord bot that uses ChatGPT (3.5-turbo) API to answer questions.

## Setup

1. Create a [Discord bot](https://discord.com/developers/applications), invite it to your server and save its token. It must be given the following permissions : `Send Messages`, `Read Messages/View Channels` and `Change Nickname` (permission byte : `67111936`).
2. Get an API key from [OpenAI](https://beta.openai.com/).
3. Remove `.dist` from `.env.dist` and fill in the values with your Discord bot token and OpenAI API key.
4. Run `go build -o ./bin` to build the bot or download it from the [releases](https://github.com/comejv/discord-gpt-bot/releases/latest) and place it in a newly created `bin` folder or in the main folder.
5. Start the bot with `./bin/gpt-bot`.

## Usage

The bot will respond to messages according to its current profile. The default profile is `condescending` which will make it answer to messages ending with `?` or that are reply to one of his messages. To change the profile, send a message : `@your-bot profile <profile-name>`.

To show the list of available profiles and commands, send a message : `@your-bot help`.