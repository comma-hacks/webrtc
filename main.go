package main

import (
	"fmt"
	"log"
	signaling "secureput"

	"github.com/commaai/cereal"
	"github.com/pion/webrtc/v3"
)

func main() {

	// trackState := make(chan bool)

	camIndex := 0
	var activeVipc *VisionIpcTrack
	var activeRtpSender *webrtc.RTPSender

	tracks := map[int]*VisionIpcTrack{}

	decoder, err := NewHEVCDecoder()
	if err != nil {
		log.Fatalln(err)
	}
	defer decoder.Close()

	encoder, err := NewH264Encoder()
	if err != nil {
		log.Fatalln(err)
	}
	defer encoder.Close()

	transcoder, err := NewVIPCTranscoder(decoder, encoder)
	if err != nil {
		log.Fatal(fmt.Errorf("main: new transcoder failed: %w", err))
	}

	for i, c := range VisionIpcCameras {
		visionTrack, err := NewVisionIpcTrack(c, *transcoder)
		if err != nil {
			log.Fatal(fmt.Errorf("main: creating track failed: %w", err))
		}
		tracks[i] = visionTrack
	}

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
		// Set the handler for ICE connection state
		// This will notify you when the peer has connected/disconnected
		pc.OnICEConnectionStateChange(func(connectionState webrtc.ICEConnectionState) {
			fmt.Printf("Connection State has changed %s \n", connectionState.String())
			if connectionState.String() == "disconnected" {
				if activeVipc != nil {
					activeVipc.Unsubscribe()
				}
			} else if connectionState == webrtc.ICEConnectionStateFailed {
				if activeVipc != nil {
					activeVipc.Unsubscribe()
				}
				if closeErr := pc.Close(); closeErr != nil {
					log.Println(fmt.Errorf("main: peer connection closed due to error: %w", closeErr))
				}
			} else if connectionState.String() == "connected" {
				if activeVipc != nil {
					activeVipc.Unsubscribe()
					activeVipc = nil
				}
				if activeRtpSender != nil {
					activeRtpSender.Stop()
					activeRtpSender = nil
				}
				selectedVipc := tracks[0]
				activeRtpSender = selectedVipc.AddTrack(pc)
				activeVipc = selectedVipc
				selectedVipc.Subscribe()
			}
		})

	}
	signal.OnMessage = func(msg signaling.Message) {
		switch msg.PayloadType {
		case "request_track":
			{
				activeVipc.Unsubscribe()
				camIndex++
				if camIndex >= len(tracks) {
					camIndex = 0
				}
				activeVipc = tracks[camIndex]
				activeVipc.Subscribe()
				activeRtpSender.ReplaceTrack(activeVipc.videoTrack)
			}
		default:
			log.Printf("unimplemented message type: %s\n", msg.PayloadType)
		}
	}

	go func() {
		for {
			if activeVipc != nil && activeRtpSender != nil {
				msgs := DrainSock(activeVipc.subscriber, false)
				msgCount := len(msgs)
				if msgCount > 0 {
					for _, msg := range msgs {
						evt, err := cereal.ReadRootEvent(msg)
						if err != nil {
							log.Println(fmt.Errorf("cereal read root event failed: %w", err))
							return
						}
						// var kind string
						// if evt.HasRoadEncodeData() {
						// 	kind = "road"
						// } else if evt.HasWideRoadEncodeData() {
						// 	kind = "wide"
						// } else if evt.HasDriverEncodeData() {
						// 	kind = "driver"
						// } else {
						// 	kind = "other"
						// }
						// fmt.Printf("%s %s\n", activeVipc.name, kind)
						activeVipc.ProcessCerealEvent(evt)

						fmt.Printf("%2d %4d %.3f %.3f roll %6.2f ms latency %6.2f ms + %6.2f ms + %6.2f ms = %6.2f ms %d %s\n",
							msgCount,
							activeVipc.encodeId,
							(float64(activeVipc.logMonoTime) / 1e9),
							(float64(activeVipc.timestampEof) / 1e6),
							activeVipc.frameLatency,
							activeVipc.processLatency,
							activeVipc.networkLatency,
							activeVipc.pcLatency,
							activeVipc.processLatency+activeVipc.networkLatency+activeVipc.pcLatency,
							activeVipc.dataSize,
							activeVipc.name,
						)
					}
				}
			}
		}
	}()

	for {
		select {}
	}
}
