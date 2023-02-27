package main

import (
	"fmt"
	"log"
	signaling "secureput"

	"github.com/pion/webrtc/v3"
)

type TrackMap map[int]*VisionIpcTrack

var camIndex int
var wantCamIndex int
var activeVipc *VisionIpcTrack
var activeRtpSender *webrtc.RTPSender
var tracks TrackMap

var Streaming chan *VisionIpcTrack

func DeactivateStream() {
	activeVipc.Unsubscribe()
}

func ActivateStream(_wantCamIndex int) {
	wantCamIndex = _wantCamIndex % len(tracks)
	if activeVipc != nil && camIndex != wantCamIndex {
		DeactivateStream()
	}
	selectedVipc := tracks[wantCamIndex]
	activeVipc = selectedVipc
	selectedVipc.Subscribe()
	camIndex = wantCamIndex
	Streaming <- activeVipc
}

func main() {
	Streaming = make(chan *VisionIpcTrack)

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
		transceiver, err := pc.AddTransceiverFromKind(webrtc.RTPCodecTypeVideo, webrtc.RtpTransceiverInit{
			Direction: webrtc.RTPTransceiverDirectionSendonly,
		})
		if err != nil {
			log.Fatal(fmt.Errorf("main: creating transceiver failed: %w", err))
		}
		activeRtpSender = transceiver.Sender()

		// Set the handler for ICE connection state
		// This will notify you when the peer has connected/disconnected
		pc.OnICEConnectionStateChange(func(connectionState webrtc.ICEConnectionState) {
			fmt.Printf("Connection State has changed %s \n", connectionState.String())
			if connectionState.String() == "disconnected" {
				DeactivateStream()
			} else if connectionState == webrtc.ICEConnectionStateFailed {
				DeactivateStream()
				if closeErr := pc.Close(); closeErr != nil {
					log.Println(fmt.Errorf("main: peer connection closed due to error: %w", closeErr))
				}
			} else if connectionState.String() == "connected" {
				ActivateStream(0)

				activeRtpSender, err = pc.AddTrack(activeVipc.videoTrack)
				if err != nil {
					log.Fatal(fmt.Errorf("main: creating track failed: %w", err))
				}

				// Later on, we will use rtpSender.ReplaceTrack() for graceful track replacement

				// Read incoming RTCP packets
				// Before these packets are returned they are processed by interceptors. For things
				// like NACK this needs to be called.
				go func() {
					rtcpBuf := make([]byte, 1500)
					for activeRtpSender != nil {
						if _, _, rtcpErr := activeRtpSender.Read(rtcpBuf); rtcpErr != nil {
							return
						}
					}
				}()
			}
		})

	}
	signal.OnMessage = func(msg signaling.Message) {
		switch msg.PayloadType {
		case "request_track":
			{
				ActivateStream(camIndex + 1)
				activeRtpSender.ReplaceTrack(activeVipc.videoTrack)
			}
		default:
			log.Printf("unimplemented message type: %s\n", msg.PayloadType)
		}
	}

	camIndex = 0

	tracks = TrackMap{}

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

	// go ActivateStream(0)

	for {
		log.Println("waitng for streaming sig")
		v := <-Streaming
		for camIndex == wantCamIndex {
			msgCount, err := v.Drain()
			if err != nil {
				log.Println(fmt.Errorf("main: vipc error: %w", err))
				continue
			}
			if msgCount < 1 {
				continue
			}
			fmt.Printf("%2d %4d %.3f %.3f roll %6.2f ms latency %6.2f ms + %6.2f ms + %6.2f ms = %6.2f ms %d %s\n",
				msgCount,
				v.encodeId,
				(float64(v.logMonoTime) / 1e9),
				(float64(v.timestampEof) / 1e6),
				v.frameLatency,
				v.processLatency,
				v.networkLatency,
				v.pcLatency,
				v.processLatency+v.networkLatency+v.pcLatency,
				v.dataSize,
				v.name,
			)
		}
	}
}
