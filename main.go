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

// Structs
type env_var struct {
	BotToken  string `json:"botToken"`
	GptApiKey string `json:"gptApiKey"`
	Log       bool   `json:"log"`
}

type stats struct {
	MsgScanned int
	Questions  int
	Answers    int
	Tokens     int
}

type profile struct {
	Name     string `json:"name"`
	Regex    string `json:"regex"`
	Context  string `json:"context"`
	Question string `json:"question"`
	Answer   string `json:"answer"`
}

// Global variables
var Env env_var
var Obs stats
var Profiles []profile
var CurrentProfile profile

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

	err = json.Unmarshal(bytes, &Env)
	if err != nil {
		fmt.Println("error unmarshalling config file")
		panic(err)
	}

	// Load profiles
	file, err = os.Open("profiles.json")
	if err != nil {
		fmt.Println("error opening profiles file")
		panic(err)
	}
	defer file.Close()

	bytes, err = io.ReadAll(file)
	if err != nil {
		fmt.Println("error reading profiles file")
		panic(err)
	}

	err = json.Unmarshal([]byte(bytes), &Profiles)
	if err != nil {
		fmt.Println("error unmarshalling profiles file")
		panic(err)
	}

	// Set the current profile
	CurrentProfile = Profiles[0]

	// Create a new Discord session using the provided bot token.
	dg, err := discordgo.New("Bot " + Env.BotToken)
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

	var err error

	// Ignore all messages created by the bot itself
	if m.Author.ID == s.State.User.ID {
		return

	} else if regexp.MustCompile(`^<@!?` + s.State.User.ID + `>`).MatchString(m.Content) {
		// If bot is tagged at the beginning of the message, update the profile
		if regexp.MustCompile(`(?mi).*profile condescending$`).MatchString(m.Content) {
			CurrentProfile = Profiles[0]
			err = (*s).GuildMemberNickname(m.GuildID, "@me", "Condescending Bot")
			fmt.Println("profile changed to condescending")
		} else if regexp.MustCompile(`(?mi)profile loving$`).MatchString(m.Content) {
			CurrentProfile = Profiles[1]
			err = (*s).GuildMemberNickname(m.GuildID, "@me", "Loving Bot")
			fmt.Println("profile changed to loving")
		} else if regexp.MustCompile(`(?mi)profile angry$`).MatchString(m.Content) {
			CurrentProfile = Profiles[2]
			err = (*s).GuildMemberNickname(m.GuildID, "@me", "Angry Bot")
			fmt.Println("profile changed to angry")
		} else if regexp.MustCompile(`(?mi)profile sad$`).MatchString(m.Content) {
			CurrentProfile = Profiles[3]
			err = (*s).GuildMemberNickname(m.GuildID, "@me", "Sad Bot")
			fmt.Println("profile changed to sad")
		} else if regexp.MustCompile(`(?mi)profile feur$`).MatchString(m.Content) {
			CurrentProfile = Profiles[4]
			err = (*s).GuildMemberNickname(m.GuildID, "@me", "Anti-feur Bot")
			fmt.Println("profile changed to feur")
		} else if regexp.MustCompile(`(?mi)profile pirate$`).MatchString(m.Content) {
			CurrentProfile = Profiles[5]
			err = (*s).GuildMemberNickname(m.GuildID, "@me", "Pirate Bot")
			fmt.Println("profile changed to pirate")
		} else if regexp.MustCompile(`(?mi)profile gamer$`).MatchString(m.Content) {
			CurrentProfile = Profiles[6]
			err = (*s).GuildMemberNickname(m.GuildID, "@me", "Gamer Bot")
			fmt.Println("profile changed to gamer")
		}
		if err != nil {
			fmt.Println("error changing nickname", err)
		}
		// Send a message to confirm the profile change
		_, err = s.ChannelMessageSend(m.ChannelID, "Profile changed to "+CurrentProfile.Name)
		if err != nil {
			fmt.Println("error sending profile change message:", err)
		}

	} else if m.Message.ReferencedMessage != nil && m.Message.ReferencedMessage.Author.ID == s.State.User.ID {
		// If the message is a reply to the bot, send a completion
		originalMessage, err := s.ChannelMessage(m.Message.ChannelID, m.Message.ReferencedMessage.ID)
		if err != nil {
			fmt.Println("error getting original message:", err)
		}

		err = send_gpt_completion(s, m, originalMessage.Content)
		if err != nil {
			fmt.Println("error sending completion:", err)
		}

	} else if regexp.MustCompile(CurrentProfile.Regex).MatchString(m.Content) {
		err = send_gpt_completion(s, m, "")
		if err != nil {
			fmt.Println("error sending completion:", err)
		}

	}
}

func send_gpt_completion(s *discordgo.Session, m *discordgo.MessageCreate, reply string) error {
	questionAsked()
	(*s).ChannelTyping(m.ChannelID)
	var answer string
	var err error

	answer, err = GptAnswer(reply, m.Author.Username, m.Content)

	if err != nil {
		fmt.Println("error getting completion:", err)
		return err
	}

	_, err = s.ChannelMessageSendReply(m.ChannelID, answer, (*m).Reference())

	if err != nil {
		fmt.Println("error sending message", err)
		return err
	}

	if Env.Log {
		fmt.Println("Message sent in ", m.GuildID, ":", m.ChannelID, ":", m.ID)
	}
	answerSent()

	return nil
}

// Observer stats
func msgScanned() {
	Obs.MsgScanned++
	if Env.Log {
		fmt.Println("Message scanned")
	}
}

func questionAsked() {
	Obs.Questions++
	if Env.Log {
		fmt.Println("Question asked")
	}
}

func answerSent() {
	Obs.Answers++
	if Env.Log {
		fmt.Println("Answer sent")
	}
}

func tokenUsed(amount int) {
	Obs.Tokens += amount
	if Env.Log {
		fmt.Println(amount, " token used")
	}
}

func printStats() {
	fmt.Println("\n======================\nStats:")
	fmt.Println("Messages scanned:", Obs.MsgScanned)
	fmt.Println("Questions asked:", Obs.Questions)
	fmt.Println("Answers sent:", Obs.Answers)
	fmt.Println("Tokens used:", Obs.Tokens)
}
