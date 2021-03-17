package main

import (
	"fmt"
	"log"
	"time"

	mordhaurcon "github.com/sniddunc/mordhau-rcon"
)

var (
	host           = "localhost"
	port     int16 = 7778
	password       = "RconPassword"
)

func main() {
	clientConfig := &mordhaurcon.ClientConfig{
		Host:              host,
		Port:              port,
		Password:          password,
		EnableBroadcasts:  true,
		BroadcastHandler:  broadcastHandler,
		DisconnectHandler: disconnectHandler,
		// SendHeartbeatCommand:     true,
		// HeartbeatCommandInterval: time.Second * 10,
	}

	client := mordhaurcon.NewClient(clientConfig)

	// Connect the main socket to the RCON server
	if err := client.Connect(); err != nil {
		log.Fatal(err)
	}

	// Create a string slice containing each broadcast type you wish to listen for.
	broadcastTypes := []string{"allon"}

	// Optional but highly recommended: create an error channel to receive errors from
	// the ListenForBroadcasts goroutine.
	errors := make(chan error)

	// Connect broadcast socket to the RCON server and start listening for broadcasts
	client.ListenForBroadcasts(broadcastTypes, nil)

	// Disconnect after 20 seconds
	go func() {
		time.Sleep(time.Second * 60)

		if err := client.Disconnect(); err != nil {
			fmt.Printf("Disconnect error: %v\n", err)
		}
	}()

	// Enter infinite loop to keep the program running. You wouldn't want to do this in practice.
	// Normally you would likely have a webserver or some other listening code you're running this
	// alongside which would keep the process running for you.
	for {
		select {
		case err := <-errors:
			log.Fatalf("ListenForBroadcasts error: %v", err)
			break
		default:
			break
		}

		// Run basic command on the main RCON socket for demo purposes.
		// response, err := client.ExecCommand("PlayerList")
		// if err != nil {
		// 	log.Fatal(err)
		// }

		// fmt.Println("Main Socket Response:", response)

		time.Sleep(10 * time.Second)
	}
}

func broadcastHandler(broadcast string) {
	fmt.Println("Received broadcast:", broadcast)
}

func disconnectHandler(err error, expected bool) {
	if !expected {
		fmt.Printf("An unexpected disconnect occurred. Error: %v\n", err)
	} else {
		fmt.Println("An expected disconnect occurred. All OK.")
	}
}
