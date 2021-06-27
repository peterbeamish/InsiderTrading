package main

import (
	"fmt"
	"os"
)

func main() {

	fmt.Println("Test123")

	hostName, _ := os.Hostname()
	fmt.Print(hostName)
}
