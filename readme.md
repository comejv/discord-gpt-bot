# Go GPT Discord bot

Discord bot that uses the GPT-3 API to answer questions.

## Setup

1. Create a Discord bot and invite it to your server.
2. Create a `.env` file with the following contents:

```json
{
    "botToken":"YOUR_BOT_TOKEN",
	"gptApiKey":"YOUR_OPENAI_API_KEY",
    "log":true
}
```
3. Run `go build -o ./bin` to build the bot.
4. Start the bot with `./bin/gptbot`.

The bot will answer any message that ends with a question mark or are replys to its own messages.
