package main

import (
	"database/sql"
	"fmt"

	"github.com/bwmarrin/discordgo"
	_ "github.com/mattn/go-sqlite3"
	openai "github.com/sashabaranov/go-openai"
)

func resetDb(db *sql.DB) {
	if Env.Log {
		fmt.Println("Resetting database...")
	}

	_, err := db.Exec("DROP TABLE IF EXISTS Users")
	if err != nil {
		fmt.Println("error resetting database")
		panic(err)
	}
	_, err = db.Exec("DROP TABLE IF EXISTS Conversations")
	if err != nil {
		fmt.Println("error resetting database")
		panic(err)
	}
}

func initDb(db *sql.DB) error {
	// Enable foreign keys
	_, err := db.Exec("PRAGMA foreign_keys = ON")
	if err != nil {
		return err
	}

	// Users
	if Env.Log {
		fmt.Println("Creating Users table...")
	}
	_, err = db.Exec("CREATE TABLE IF NOT EXISTS Users (uId VARCHAR(20) NOT NULL PRIMARY KEY, " +
		"uNick VARCHAR(32))")
	if err != nil {
		return err
	}

	// Messages
	if Env.Log {
		fmt.Println("Creating Messages table...")
	}
	_, err = db.Exec("CREATE TABLE IF NOT EXISTS Conversations (mId VARCHAR(20) NOT NULL PRIMARY KEY, " +
		"uId VARCHAR(20), mContent TEXT, mAnswer TEXT, mChannel VARCHAR(20), " +
		"FOREIGN KEY (uId) REFERENCES Users(uId))")
	if err != nil {
		return err
	}

	return nil
}

func insertUser(db *sql.DB, uId string, uNick string) {
	if Env.Log {
		fmt.Println("Inserting user " + uNick + " (" + uId + ")")
	}
	_, err := db.Exec("INSERT OR IGNORE INTO Users (uId, uNick) VALUES (?, ?)", uId, uNick)
	if err != nil {
		panic(err)
	}
}

func insertMessage(db *sql.DB, m *discordgo.MessageCreate) {
	if Env.Log {
		fmt.Println("Inserting message" + " (" + m.ID + ") from user " + m.Author.Username + " (" + m.Author.ID + ")")
	}
	mId := m.ID
	mContent := m.Content
	mAuthorId := m.Author.ID
	mChannel := m.ChannelID
	_, err := db.Exec("INSERT INTO Conversations (mId, uId, mContent, mAnswer, mChannel) VALUES (?, ?, ?, ?, ?)", mId, mAuthorId, mContent, "", mChannel)
	if err != nil {
		panic(err)
	}
}

func insertAnswer(db *sql.DB, mId string, mAnswer string) {
	if Env.Log {
		fmt.Println("Inserting answer to message " + mId)
	}
	_, err := db.Exec("UPDATE Conversations SET mAnswer = ? WHERE mId = ?", mAnswer, mId)
	if err != nil {
		panic(err)
	}
}

func updateUser(db *sql.DB, uId string, uNick string) {
	_, err := db.Exec("UPDATE Users SET uNick = ? WHERE uId = ?", uNick, uId)
	if err != nil {
		panic(err)
	}
}

func getConversations(db *sql.DB, uId string) []openai.ChatCompletionMessage {
	// Get author nick
	var uNick string
	err := db.QueryRow("SELECT uNick FROM Users WHERE uId = ?", uId).Scan(&uNick)
	if err != nil {
		panic(err)
	}

	// Get messages
	rows, err := db.Query("SELECT mContent, mAnswer FROM Conversations WHERE uId = ?", uId)
	if err != nil {
		panic(err)
	}
	defer rows.Close()

	var messages []openai.ChatCompletionMessage

	s := openai.ChatCompletionMessage{
		Role:    "system",
		Content: "Tu es un bot discord. La personne qui te parle actuellement est " + uNick + ".\n" + CurrentProfile.Context}

	messages = append(messages, s)

	for rows.Next() {
		var mContent string
		var mAnswer string
		err = rows.Scan(&mContent, &mAnswer)
		if err != nil {
			panic(err)
		}
		q := openai.ChatCompletionMessage{Role: "user", Content: mContent}
		a := openai.ChatCompletionMessage{Role: "assistant", Content: mAnswer}
		messages = append(messages, q)
		if a.Content != "" {
			messages = append(messages, a)
		}
	}

	err = rows.Err()
	if err != nil {
		panic(err)
	}

	return messages
}

func numberMessagesUser(db *sql.DB, uId string) string {
	var mId string
	err := db.QueryRow("SELECT mId FROM Conversations WHERE uId = ? ORDER BY mId DESC LIMIT 1", uId).Scan(&mId)
	if err != sql.ErrNoRows && err != nil {
		panic(err)
	}
	return mId
}

func clearUserMessages(db *sql.DB, uId string) {
	if Env.Log {
		fmt.Println("Clearing messages from user " + uId)
	}
	_, err := db.Exec("DELETE FROM Conversations WHERE uId = ?", uId)
	if err != nil {
		panic(err)
	}
}
