package main

import (
	"MartellX/discord_bot/discord"
	"fmt"
	"os"
	"os/signal"
	"syscall"
)

func main() {
	discord.Init()
	fmt.Println("YoBot is running")
	fmt.Println("Press CTRL-C to exit.")
	sc := make(chan os.Signal, 1)
	signal.Notify(sc, syscall.SIGINT, syscall.SIGTERM, os.Interrupt, os.Kill)
	<-sc

	// Cleanly close down the Discord session.
	discord.Close()
}
