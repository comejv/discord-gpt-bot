package main

import (
	"encoding/json"
	"errors"
	"io"
	"log"
	"net/http"
	"strings"
)

const (
	apiURL = "https://api.openai.com/v1/completions"
)

type gptQuery struct {
	Model       string  `json:"model"`
	Prompt      string  `json:"prompt"`
	Max_tokens  int     `json:"max_tokens"`
	Temperature float64 `json:"temperature"`
	Stop        string  `json:"stop"`
}

func GptAnswer(reply string, question string, name string) (string, error) {
	// Create a new HTTP client
	client := &http.Client{}

	var prompt string

	// Create the prompt
	if reply == "" {
		prompt = "Tu es un bot discord qui répond de manière condescendante aux questions." +
			"\nTu n'hésites pas à te plaindre quand il manque de politesse." +
			"\nQuestion de " + name + ": " +
			question +
			"\nRéponse condescendante : "

	} else {

		prompt = "Tu es un bot discord qui répond de manière condescendante au questions." +
			"\nTu n'hésites pas te plaindre quand il manque de politesse." +
			"\nMessage : [...]" +
			"\nRéponse condescendante : " +
			reply +
			"\nMessage de " + name + ": " +
			question +
			"\nRéponse condescendante : "
	}

	// Create the query
	qbytes, _ := json.Marshal(gptQuery{
		Model:       "text-davinci-003",
		Prompt:      prompt,
		Max_tokens:  100,
		Temperature: 0.5,
		Stop:        "Question : ",
	})

	// Create a new POST request
	req, err := http.NewRequest("POST", apiURL, strings.NewReader(string(qbytes)))

	if err != nil {
		return "", err
	}

	// Set the Authorization header
	req.Header.Set("Authorization", "Bearer " + env.GptApiKey)

	// Set the Content-Type header
	req.Header.Set("Content-Type", "application/json")

	// Send the request and get the response
	resp, err := client.Do(req)
	if err != nil {
		log.Fatal(err)
	}
	defer resp.Body.Close()

	// Check the response status code
	if resp.StatusCode != http.StatusOK {
		return "", errors.New("gpt: bad status code")
	}

	// Read the response body
	rbytes, err := io.ReadAll(resp.Body)
	if err != nil {
		log.Fatal(err)
	}

	// Unmarshal the response into a map
	var body map[string]interface{}
	err = json.Unmarshal(rbytes, &body)
	if err != nil {
		log.Fatal(err)
	}

	// Get the value of the text field from the first choice
	choices, ok := body["choices"].([]interface{})
	if !ok {
		log.Fatal("choices field not found or not an array")
	}

	choice, ok := choices[0].(map[string]interface{})
	if !ok {
		log.Fatal("first choice not found or not an object")
	}

	text, ok := choice["text"].(string)
	if !ok {
		log.Fatal("text field not found or not a string")
	}

	usage := body["usage"].(map[string]interface{})
	tokenUsed(int(usage["total_tokens"].(float64)))

	// Return the response body
	return text, nil
}
