package main

import (
	"fmt"
	"os"
)

func main() {
	fmt.Println("Starting Chatbot...")
	// Load Configurations
	apiKey := os.Getenv("API_KEY")
	if apiKey == "" {
		fmt.Println("API key not found. Please set it in the .env file.")
		return
	}

	// Simulate a simple chatbot interaction
	fmt.Println("Chatbot: Hi! How can I assist you today?")
	// For simplicity, this chatbot does not process input, just prints a response
	fmt.Println("Chatbot: Thank you for your question! We'll get back to you soon.")
}