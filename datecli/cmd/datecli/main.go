package main

import (
	"fmt"
	"time"
)

func main() {
	now := time.Now()
	fmt.Printf("Current date and time: %s\n", now.Format(time.RFC1123))
}
