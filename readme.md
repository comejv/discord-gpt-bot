# Go GPT Discord bot

Discord bot that uses the GPT-3 API to answer questions.

## Setup

1. Create a Discord bot and invite it to your server and save its token.
2. Get an API key from [OpenAI](https://beta.openai.com/).
3. Remove `.dist` from `.env.dist` and fill in the values with your Discord bot token and GPT-3 API key.
4. Run `go build -o ./bin` to build the bot.
5. Start the bot with `./bin/gptbot`.

## Usage

The bot will respond to messages according to its current profile. The default profile is `condescending` which will make it answer to messages ending with `?`. To change the profile, send a message : `@your-bot profile <profile-name>`.

To show the list of available profiles and commands, send a message : `@your-bot help`.