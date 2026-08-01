package handlers

import "fmt"

func HandleQuery(query string) {
	fmt.Printf("Received query: %s\n", query)
	fmt.Println("Processing via AI...")
	// Simulate AI processing
	fmt.Println("Response: That's an interesting question!")
}