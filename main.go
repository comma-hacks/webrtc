package main

import (
	"fmt"
	"log"
	signaling "secureput"

	"github.com/pion/webrtc/v3"
)

var visionTrack *VisionIpcTrack
var peerConnection *webrtc.PeerConnection

func main() {
	signal := signaling.Create("go-webrtc-body")
	signal.DeviceMetadata = map[string]interface{}{"isRobot": true}
	signal.Gui = &Face{app: &signal}
	go signal.RunDaemonMode()
	if !signal.Paired() {
		go signal.Gui.Show()
		log.Println("Waiting to pair.")
		<-signal.PairWaitChannel
	}

	signal.OnPeerConnectionCreated = func(pc *webrtc.PeerConnection) {
		peerConnection = pc
		InitStream(pc)
	}
	signal.OnMessage = func(msg signaling.Message) {
		switch msg.PayloadType {
		case "request_track":
			{
				visionTrack.ChangeCamera()
			}
		default:
			log.Printf("unimplemented message type: %s\n", msg.PayloadType)
		}
	}
	for {
		select {}
	}
}

func InitStream(peerConnection *webrtc.PeerConnection) {
	var err error
	if visionTrack != nil {
		visionTrack.Stop()
	}
	visionTrack, err = NewVisionIpcTrack()
	if err != nil {
		log.Fatal(fmt.Errorf("main: creating track failed: %w", err))
	}

	rtpSender, err := peerConnection.AddTrack(visionTrack.videoTrack)
	if err != nil {
		log.Fatal(fmt.Errorf("main: creating track failed: %w", err))
	}

	_, err = peerConnection.AddTransceiverFromTrack(visionTrack.videoTrack,
		webrtc.RtpTransceiverInit{
			Direction: webrtc.RTPTransceiverDirectionSendonly,
		},
	)
	if err != nil {
		log.Fatal(fmt.Errorf("main: creating transceiver failed: %w", err))
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
			go visionTrack.Start()
		}
	})
}
