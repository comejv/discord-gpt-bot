# Go GPT Discord bot

Discord bot that uses the GPT-3 API to answer questions.

## Setup

1. Create a Discord bot and invite it to your server.
2. Remove `.dist` from `.env.dist` and fill in the values with your Discord bot token and GPT-3 API key.
3. Run `go build -o ./bin` to build the bot.
4. Start the bot with `./bin/gptbot`.

The bot will answer any message that ends with a question mark or are replys to its own messages.

Note: The bot is currently set to be condescending. You can change this by changing the prompt in [gpt-api.go](gpt-api.go)
