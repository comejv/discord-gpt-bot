package main

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"os/signal"
	"path/filepath"
	"regexp"
	"strconv"
	"syscall"
	"time"

	"github.com/bwmarrin/discordgo"
	_ "github.com/mattn/go-sqlite3"
	openai "github.com/sashabaranov/go-openai"
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
	Profiles   int
}

type profile struct {
	Name     string `json:"name"`
	Regex    string `json:"regex"`
	Context  string `json:"context"`
	Question string `json:"question"`
	Answer   string `json:"answer"`
}

type last struct {
	Guild   string
	Channel string
}

// Global variables
var (
	Env            env_var
	Obs            stats
	Profiles       []profile
	CurrentProfile profile
	LastGuildChan  last
	db             *sql.DB
	startTime      time.Time
	client         *openai.Client
)

func main() {
	// Load environment variables
	execPath, err := os.Executable()
	if err != nil {
		panic(err)
	}

	// Check if the bot is running from the main folder or the binaries folder
	var baseDir string
	if filepath.Base(filepath.Dir(execPath)) == "bin" {
		baseDir = filepath.Dir(filepath.Dir(execPath)) + "/"
	} else {
		baseDir = filepath.Dir(execPath) + "/"
	}

	// Load config
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

	// Create/reset database
	db, err = sql.Open("sqlite3", "file:"+baseDir+"data/database.db?mode=rwc")
	if err != nil {
		fmt.Println("error opening/creating database")
		panic(err)
	}
	defer db.Close()

	// Initialize the database
	// Ask user if they want to reset the database
	var reset string
	fmt.Print("Do you want to reset the database? (y/n) ")
	fmt.Scanln(&reset)
	if reset == "y" {
		resetDb(db)
	}

	err = initDb(db)
	if err != nil {
		fmt.Println("error initializing database")
		panic(err)
	}

	// Initiate openai client
	client = openai.NewClient(Env.GptApiKey)

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
	// Get current time
	startTime = time.Now()

	// Wait here until CTRL-C or other term signal is received.
	fmt.Println("Bot is now running. Press CTRL-C to exit.")
	sc := make(chan os.Signal, 1)
	signal.Notify(sc, syscall.SIGINT, syscall.SIGTERM, os.Interrupt)
	<-sc

	// Reset username and send a goodbye message in the last channel the bot was active in
	err = (*dg).GuildMemberNickname(LastGuildChan.Guild, "@me", "")
	if err != nil {
		dg.ChannelMessageSend(LastGuildChan.Channel, "Error resetting nickname! Please reset it manually. :(")
		fmt.Println("error changing nickname", err)
	}
	_, err = dg.ChannelMessageSend(LastGuildChan.Channel, "Bot shutting down for maintenance!")
	if err != nil {
		fmt.Println("error sending goodbye message:", err)
	}
	_, err = dg.ChannelMessageSend(LastGuildChan.Channel, "Stats :\n```\n"+getStats(startTime)+"\n```")
	if err != nil {
		fmt.Println("error sending stats:", err)
	}

	// Cleanly close down the Discord session.
	dg.Close()

	// Print stats
	fmt.Print(getStats(startTime))
}

func messageCreate(s *discordgo.Session, m *discordgo.MessageCreate) {
	msgScanned()

	var err error

	// Ignore all messages created by the bot itself
	if m.Author.ID == s.State.User.ID {
		// Update the current channel
		LastGuildChan.Guild = m.GuildID
		LastGuildChan.Channel = m.ChannelID
		return

	} else if isMentioned(m.Mentions, s.State.User) {
		// If bot is tagged in the message, do command

		// profile command
		if regexp.MustCompile(`(?mi)^.*profile`).MatchString(m.Content) {
			// Find the profile name
			needle := regexp.MustCompile(`(?i)profile\s+(?P<name>\w+)`).FindStringSubmatch(m.Content)
			changed := false

			// Find the corresponding profile
			for _, profile := range Profiles {
				if profile.Name == needle[1] {
					CurrentProfile = profile
					changed = true
					profileChanged()
					err = (*s).GuildMemberNickname(m.GuildID, "@me", profile.Name)
					if err != nil {
						fmt.Println("error changing nickname", err)
					}
					fmt.Println("profile changed to", profile.Name)
					break
				}
			}

			// Send a message to confirm the profile change if no errors occured
			if changed {
				_, err = s.ChannelMessageSend(m.ChannelID, "Profile changed to "+CurrentProfile.Name)
				if err != nil {
					fmt.Println("error sending profile change message:", err)
				}
				answerSent()
			} else {
				_, err = s.ChannelMessageSend(m.ChannelID, "Profile not found")
				if err != nil {
					fmt.Println("error sending profile not found message:", err)
				}
			}

			return
		}

		// stats command
		if regexp.MustCompile(`(?i)stats$`).MatchString(m.Content) {
			// Create a stat message
			msg := "```\n" + getStats(startTime) + "\n```"
			// Send a message with the stats
			_, err = s.ChannelMessageSend(m.ChannelID, msg)
			if err != nil {
				fmt.Println("error sending stats message:", err)
			}
			return
		}

		// help command
		if regexp.MustCompile(`(?i)help$`).MatchString(m.Content) {
			// Create a help message
			msg := "```"
			msg += "Commands:"
			msg += "\n@Smartass profile <name>: change the profile"
			msg += "\n@Smartass stats: show statistics"
			msg += "\n@Smartass help: show this message"
			msg += "\n\nProfiles:"
			for _, profile := range Profiles {
				msg += "\n- " + profile.Name + ": " + profile.Regex
			}
			msg += "```"
			// Send a message with the help
			_, err = s.ChannelMessageSend(m.ChannelID, msg)
			if err != nil {
				fmt.Println("error sending help message:", err)
			}
			return
		}
	}

	// If the message is a reply to the bot, send a completion
	if m.ReferencedMessage != nil && m.ReferencedMessage.Author.ID == s.State.User.ID {
		// Update DB
		if messagesTimeDiff(m.ID, numberMessagesUser(db, m.Author.ID)) > 5*time.Minute {
			clearUserMessages(db, m.Author.ID)
		}
		insertUser(db, m.Author.ID, m.Author.Username)
		insertMessage(db, m)

		originalMessage, err := s.ChannelMessage(m.ChannelID, m.ReferencedMessage.ID)
		if err != nil {
			fmt.Println("error getting original message:", err)
		}

		err = sendChatCompletion(s, m, originalMessage.Content)
		if err != nil {
			fmt.Println("error sending completion:", err)
		}

	} else if regexp.MustCompile(CurrentProfile.Regex).MatchString(m.Content) {
		// Update DB
		if messagesTimeDiff(m.ID, numberMessagesUser(db, m.Author.ID)) > 5*time.Minute {
			clearUserMessages(db, m.Author.ID)
		}
		insertUser(db, m.Author.ID, m.Author.Username)
		insertMessage(db, m)

		err = sendChatCompletion(s, m, "")
		if err != nil {
			fmt.Println("error sending completion:", err)
		}

	}
}

func isMentioned(s []*discordgo.User, u *discordgo.User) bool {
	for _, m := range s {
		if m.ID == u.ID {
			return true
		}
	}

	return false
}

func snowflakeToUnix(s string) time.Time {
	si, err := strconv.Atoi(s)
	if err != nil {
		fmt.Println("error converting snowflake to int:", err)
		panic(err)
	}
	return time.UnixMilli(int64(si>>22 + 1420070400000))
}

func messagesTimeDiff(m1 string, m2 string) time.Duration {
	if m1 == "" || m2 == "" {
		return 0
	}
	return snowflakeToUnix(m1).Sub(snowflakeToUnix(m2))
}

func sendChatCompletion(s *discordgo.Session, m *discordgo.MessageCreate, reply string) error {
	questionAsked()
	(*s).ChannelTyping(m.ChannelID)
	var err error

	// Get previous messages
	messages := getConversations(db, m.Author.ID)

	// Create completion request
	request := openai.ChatCompletionRequest{
		Model:    openai.GPT3Dot5Turbo,
		Messages: messages,
	}

	// Get completion
	completion, err := client.CreateChatCompletion(
		context.Background(),
		request)

	// Update tokens
	tokenUsed(completion.Usage.TotalTokens)

	// Extract completion
	answer := completion.Choices[0].Message.Content

	// Update conversation
	insertAnswer(db, m.ID, answer)

	if err != nil {
		fmt.Println("error getting completion:", err)
		return err
	}

	_, err = s.ChannelMessageSendReply(m.ChannelID, answer, (*m).Reference())
	if err != nil {
		fmt.Println("error sending message", err)
		return err
	}

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

func profileChanged() {
	Obs.Profiles++
	if Env.Log {
		fmt.Println("Profile changed")
	}
}

func getStats(t time.Time) string {
	msg := "Uptime : " + time.Since(t).Round(time.Second).String()
	msg += "\nMessages scanned: " + strconv.Itoa(Obs.MsgScanned)
	msg += "\nQuestions asked: " + strconv.Itoa(Obs.Questions)
	msg += "\nAnswers sent: " + strconv.Itoa(Obs.Answers)
	msg += "\nTokens used: " + strconv.Itoa(Obs.Tokens)
	msg += "\nProfiles change: " + strconv.Itoa(Obs.Profiles)
	msg += "\n"
	return msg
}
