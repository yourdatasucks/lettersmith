package main

import (
	"fmt"
	"log"
	"os"
)

func main() {
	if len(os.Args) < 2 {
		log.Fatal("Usage: migrate [up|down]")
	}

	command := os.Args[1]
	
	switch command {
	case "up":
		fmt.Println("Running migrations UP...")
		fmt.Println("Would apply migrations from migrations/ directory")
	case "down":
		fmt.Println("Running migrations DOWN...")
		fmt.Println("Would rollback migrations")
	default:
		log.Fatalf("Unknown command: %s", command)
	}
} 