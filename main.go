package main

import (
	"fmt"
	"log"
	"secureput"

	"github.com/pion/webrtc/v3"
)

var visionTrack *VisionIpcTrack

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
		// ReplaceTrack("road", pc)
	}

	visionTrack, _ = NewVisionIpcTrack("roadEncodeData")
	videoTrack, _ := visionTrack.NewTrackRTP()
	visionTrack.StartRTP(videoTrack)

	for {
		select {}
	}
}

func ReplaceTrack(prefix string, peerConnection *webrtc.PeerConnection) {
	var err error
	if visionTrack != nil {
		visionTrack.Stop()
	}
	visionTrack, err = NewVisionIpcTrack(prefix + "EncodeData")
	if err != nil {
		log.Fatal(fmt.Errorf("main: creating track failed: %w", err))
	}

	videoTrack, err := visionTrack.NewTrackRTP()
	if err != nil {
		log.Fatal(fmt.Errorf("main: creating track failed: %w", err))
	}

	rtpSender, err := peerConnection.AddTrack(videoTrack)
	if err != nil {
		log.Fatal(fmt.Errorf("main: creating track failed: %w", err))
	}
	// Later on, we will use rtpSender.ReplaceTrack() for graceful track replacement

	// Read incoming RTCP packets
	// Before these packets are returned they are processed by interceptors. For things
	// like NACK this needs to be called.
	go func() {
		rtcpBuf := make([]byte, 1500)
		for {
			if _, _, rtcpErr := rtpSender.Read(rtcpBuf); rtcpErr != nil {
				return
			}
		}
	}()

	// Set the handler for ICE connection state
	// This will notify you when the peer has connected/disconnected
	peerConnection.OnICEConnectionStateChange(func(connectionState webrtc.ICEConnectionState) {
		fmt.Printf("Connection State has changed %s \n", connectionState.String())
		if connectionState.String() == "disconnected" {
			visionTrack.Stop()
		} else if connectionState == webrtc.ICEConnectionStateFailed {
			visionTrack.Stop()
			if closeErr := peerConnection.Close(); closeErr != nil {
				log.Println(fmt.Errorf("main: peer connection closed due to error: %w", err))
			}
		} else if connectionState.String() == "connected" {
			go visionTrack.StartRTP(videoTrack)
		}
	})
}
