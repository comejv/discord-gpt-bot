# Go GPT Discord bot

Discord bot that uses the GPT-3 API to answer questions.

## Setup

1. Create a Discord bot and invite it to your server.
2. In [/data](data), remove `.dist` from `.env.dist` and fill in the values with your Discord bot token and GPT-3 API key.
3. Run `go build -o ./bin` to build the bot.
4. Start the bot with `./bin/gptbot`.

## Usage

The bot will respond to messages according to its current profile. The default profile is `condescending` which will make it answer to messages ending with `?`. To change the profile, use the `profile` command.