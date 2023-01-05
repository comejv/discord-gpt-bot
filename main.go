package main

import (
	"encoding/json"
	"fmt"
	"io"
	"os"
	"os/signal"
	"path/filepath"
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
var CurrentChannel string

func main() {
	// Load environment variables
	execPath, err := os.Executable()
	if err != nil {
		panic(err)
	}

	baseDir := filepath.Dir(filepath.Dir(execPath)) + "/"

	file, err := os.Open(baseDir + ".env")
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
	file, err = os.Open(baseDir + "data/profiles.json")
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

	// Send a goodbye message in the last channel the bot was active in
	_, err = dg.ChannelMessageSend(CurrentChannel, "Bot shutting down for maintenance!")
	if err != nil {
		fmt.Println("error sending goodbye message:", err)
	}

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
		if regexp.MustCompile(`(?mi)^.*profile`).MatchString(m.Content) {
			// Find the profile name
			needle := regexp.MustCompile(`(?i)profile\s+(?P<name>\w+)`).FindStringSubmatch(m.Content)

			// Find the corresponding profile
			for _, profile := range Profiles {
				if profile.Name == needle[1] {
					CurrentProfile = profile
					err = (*s).GuildMemberNickname(m.GuildID, "@me", profile.Name)
					if err != nil {
						fmt.Println("error changing nickname", err)
					}
					fmt.Println("profile changed to", profile.Name)
					break
				}
			}

			// Send a message to confirm the profile change
			_, err = s.ChannelMessageSend(m.ChannelID, "Profile changed to "+CurrentProfile.Name)
			if err != nil {
				fmt.Println("error sending profile change message:", err)
			}
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

	// Update the current channel
	CurrentChannel = m.ChannelID

	// Update stats
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
