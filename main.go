package main

import (
	"fmt"
	"log"
	"secureput"

	"github.com/pion/webrtc/v3"
)

var vipctrack *VisionIpcTrack

func StopTrack() {
	if vipctrack != nil {
		vipctrack.Stop()
		vipctrack = nil
	}
}

func ReplaceTrack(prefix string, pc *webrtc.PeerConnection) {
	StopTrack()
	var err error
	vipctrack, err = NewVisionIpcTrack(prefix + "EncodeData")
	if err != nil {
		log.Fatal(fmt.Errorf("main: creating track failed: %w", err))
	}
	go vipctrack.Start()

	webrtc.newtrack
	for frame := range vipctrack.Frame {
		// Do something with decoded frame
	}
}

func main() {
	signal := secureput.Create("go-webrtc-body")
	signal.DeviceMetadata = map[string]interface{}{"isRobot": true}
	signal.Gui = &Face{app: &signal}
	go signal.RunDaemonMode()
	if !signal.Paired() {
		go signal.Gui.Show()
		log.Println("Waiting to pair.")
		<-signal.PairWaitChannel
	}
	signal.OnPeerConnectionCreated = func(pc *webrtc.PeerConnection) {
		ReplaceTrack("road", pc)
	}
	signal.OnICEConnectionStateChange = func(connectionState webrtc.ICEConnectionState) {
		if connectionState.String() == "disconnected" {
			StopTrack()
		}
	}
	for {
	}
}
