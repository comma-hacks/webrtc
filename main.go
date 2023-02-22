package main

import (
	"log"
	"secureput"
)

func main() {
	signal := secureput.Create("go-webrtc-body")
	signal.Gui = &Face{app: &signal}
	go signal.RunDaemonMode()

	if !signal.Paired() {
		go signal.Gui.Show()
		log.Println("Waiting to pair.")
		<-signal.PairWaitChannel
	}

	// go TestVisionIPCTrack("roadEncodeData")

	for {
		select {}
	}

}
