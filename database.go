package main

import (
	"database/sql"
	"fmt"

	"github.com/bwmarrin/discordgo"
	_ "github.com/mattn/go-sqlite3"
	gogpt "github.com/sashabaranov/go-gpt3"
)

func init_db(db *sql.DB) error {

	// Users
	fmt.Println("Creating Users table...")
	_, err := db.Exec("CREATE TABLE IF NOT EXISTS Users (uId VARCHAR(20) NOT NULL PRIMARY KEY, " +
		"uNick VARCHAR(32))")
	if err != nil {
		return err
	}

	// Messages
	fmt.Println("Creating Messages table...")
	_, err = db.Exec("CREATE TABLE IF NOT EXISTS Messages (mId VARCHAR(20) NOT NULL PRIMARY KEY, " +
		"mContent TEXT, uId VARCHAR(20), mChannel VARCHAR(20), " +
		"FOREIGN KEY (uId) REFERENCES Users(uId))")
	if err != nil {
		return err
	}

	return nil
}

func insert_user(db *sql.DB, uId string, uNick string) {
	if Env.Log {
		fmt.Println("Inserting user " + uNick + " (" + uId + ")")
	}
	_, err := db.Exec("INSERT OR IGNORE INTO Users (uId, uNick) VALUES (?, ?)", uId, uNick)
	if err != nil {
		panic(err)
	}
}

func insert_message(db *sql.DB, m *discordgo.MessageCreate) {
	if Env.Log {
		fmt.Println("Inserting message from " + m.Author.Username + " (" + m.ID + ")")
	}
	mId := m.ID
	mContent := m.Content
	mAuthor := m.Author.ID
	mChannel := m.ChannelID
	_, err := db.Exec("INSERT INTO Messages (mId, mContent, uId, mChannel) VALUES (?, ?, ?, ?)", mId, mContent, mAuthor, mChannel)
	if err != nil {
		panic(err)
	}
}

func update_user(db *sql.DB, uId string, uNick string) {
	_, err := db.Exec("UPDATE Users SET uNick = ? WHERE uId = ?", uNick, uId)
	if err != nil {
		panic(err)
	}
}

func get_messages_content(db *sql.DB, uId string) []gogpt.ChatCompletionMessage {
	// Get author nick
	var uNick string
	err := db.QueryRow("SELECT uNick FROM Users WHERE uId = ?", uId).Scan(&uNick)
	if err != nil {
		panic(err)
	}

	// Get messages
	rows, err := db.Query("SELECT mContent FROM Messages WHERE uId = ?", uId)
	if err != nil {
		panic(err)
	}
	defer rows.Close()

	var messages []gogpt.ChatCompletionMessage
	for rows.Next() {
		var mContent string
		err = rows.Scan(&mContent)
		if err != nil {
			panic(err)
		}
		messages = append(messages, gogpt.ChatCompletionMessage{Role: uNick, Content: mContent})
	}

	err = rows.Err()
	if err != nil {
		panic(err)
	}

	return messages
}

func get_previous_message(db *sql.DB, uId string) string {
	var mId string
	err := db.QueryRow("SELECT mId FROM Messages WHERE uId = ? ORDER BY mId DESC LIMIT 1", uId).Scan(&mId)
	if err != sql.ErrNoRows && err != nil {
		panic(err)
	}
	return mId
}

func clear_user_messages(db *sql.DB, uId string) {
	if Env.Log {
		fmt.Println("Clearing messages from user " + uId)
	}
	_, err := db.Exec("DELETE FROM Messages WHERE uId = ?", uId)
	if err != nil {
		panic(err)
	}
}
