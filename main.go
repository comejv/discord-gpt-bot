package main

import (
	"encoding/json"
	"fmt"
	"io"
	"os"
	"os/signal"
	"regexp"
	"syscall"

	"github.com/bwmarrin/discordgo"
)

type env_var struct {
	BotToken  string `json:"botToken"`
	GptApiKey string `json:"gptApiKey"`
	Log		  bool `json:"log"`
}

type stats struct {
	MsgScanned int
	Questions  int
	Answers    int
	Tokens     int
}

// Global variables
var env env_var
var obs stats

func main() {
	// Load environment variables
	file, err := os.Open(".env")
	if err != nil {
		fmt.Println("error opening config file")
		panic(err)
	}
	defer file.Close()

	bytes, err := io.ReadAll(file)
	if err != nil {
		fmt.Println("error reading config file")
		panic(err)
	}

	err = json.Unmarshal(bytes, &env)
	if err != nil {
		fmt.Println("error unmarshalling config file")
		panic(err)
	}

	// Create a new Discord session using the provided bot token.
	dg, err := discordgo.New("Bot " + env.BotToken)
	if err != nil {
		fmt.Println("error creating Discord session")
		panic(err)
	}

	// Register the messageCreate func as a callback for MessageCreate events.
	dg.AddHandler(messageCreate)
	dg.Identify.Intents = discordgo.IntentsGuildMessages

	// Open a websocket connection to Discord and begin listening.
	err = dg.Open()
	if err != nil {
		fmt.Println("error opening connection")
		panic(err)
	}
	defer printStats()

	// Wait here until CTRL-C or other term signal is received.
	fmt.Println("Bot is now running. Press CTRL-C to exit.")
	sc := make(chan os.Signal, 1)
	signal.Notify(sc, syscall.SIGINT, syscall.SIGTERM, os.Interrupt)
	<-sc

	// Cleanly close down the Discord session.
	dg.Close()
}

func messageCreate(s *discordgo.Session, m *discordgo.MessageCreate) {
	msgScanned()

	// Ignore all messages created by the bot itself
	if m.Author.ID == s.State.User.ID {
		return
	}

	if m.Message.ReferencedMessage != nil && m.Message.ReferencedMessage.Author.ID == s.State.User.ID {
		questionAsked()

		originalMessage, err := s.ChannelMessage(m.Message.ChannelID, m.Message.ReferencedMessage.ID)
		if err != nil {
			fmt.Println("error getting original message:", err)
		}

		answer, err := GptAnswer(originalMessage.Content, m.Content, m.Author.Username)

		if err != nil {
			fmt.Println("error getting completion:", err)
			return
		}

		_, err = s.ChannelMessageSend(m.ChannelID, answer)

		if err != nil {
			fmt.Println("error sending message")
			return
		}

		answerSent()

	// If the message ends with '?', send a completion
	} else if regexp.MustCompile(`\?$`).MatchString(m.Content) {
		questionAsked()

		var answer string
		var err error

		// If message is a reply, get the original message too
		answer, err = GptAnswer("", m.Content, m.Author.Username)

		if err != nil {
			fmt.Println("error getting completion:", err)
			return
		}

		_, err = s.ChannelMessageSend(m.ChannelID, answer)

		if err != nil {
			fmt.Println("error sending message")
			return
		}

		answerSent()
	}
}

// Observer stats
func msgScanned() {
	obs.MsgScanned++
	if env.Log {
		fmt.Println("Message scanned")
	}
}

func questionAsked() {
	obs.Questions++
	if env.Log {
		fmt.Println("Question asked")
	}
}

func answerSent() {
	obs.Answers++
	if env.Log {
		fmt.Println("Answer sent")
	}
}

func tokenUsed(amount int) {
	obs.Tokens += amount
	if env.Log {
		fmt.Println(amount, " token used")
	}
}

func printStats() {
	fmt.Println("\n======================\nStats:")
	fmt.Println("Messages scanned:", obs.MsgScanned)
	fmt.Println("Questions asked:", obs.Questions)
	fmt.Println("Answers sent:", obs.Answers)
	fmt.Println("Tokens used:", obs.Tokens)
}
