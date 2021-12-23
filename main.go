package main

import (
	"fmt"
	"github.com/joho/godotenv"
	"log"
	"os"
	"os/signal"
	"strconv"
	"sync"
	"syscall"
)

func main() {
	log.Println("[Main] Starting up Discord Plays website")

	err := godotenv.Load()
	if err != nil {
		log.Fatal("Error loading .env file")
	}

	httpPort, err := strconv.Atoi(os.Getenv("PORT"))
	if err != nil {
		log.Fatal("Error getting PORT")
	}

	wg := &sync.WaitGroup{}
	wg.Add(1)
	dpHttp := &DiscordPlaysHttp{}

	//=====================
	// Safe shutdown
	sigs := make(chan os.Signal, 1)
	signal.Notify(sigs, syscall.SIGINT, syscall.SIGTERM)

	go func() {
		<-sigs
		fmt.Printf("\n")
		log.Printf("[Main] Attempting safe shutdown\n")
		dpHttp.Shutdown()
		wg.Done()
	}()
	//
	//=====================

	dpHttp.StartupHttp(httpPort, wg)

	// Wait for exit
	log.Printf("[Main] Waiting for close signal\n")
	wg.Wait()
	log.Printf("[Main] Goodbye\n")
}
