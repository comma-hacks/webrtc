package main

import (
	"fmt"
	"log"
	signaling "secureput"
	"strings"

	"github.com/asticode/go-astiav"
	"github.com/pion/webrtc/v3"
)

type TrackMap map[int]*VisionIpcTrack

var camIndex int
var wantCamIndex int
var activeVipc *VisionIpcTrack
var activeRtpSender *webrtc.RTPSender
var tracks TrackMap
var err error

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
		// transceiver, err := pc.AddTransceiverFromKind(webrtc.RTPCodecTypeVideo, webrtc.RtpTransceiverInit{
		// 	Direction: webrtc.RTPTransceiverDirectionSendonly,
		// })
		// if err != nil {
		// 	log.Fatal(fmt.Errorf("main: creating transceiver failed: %w", err))
		// }
		// activeRtpSender = transceiver.Sender()

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

				// transceiver, err := pc.AddTransceiverFromKind(webrtc.RTPCodecTypeVideo, webrtc.RtpTransceiverInit{
				// 	Direction: webrtc.RTPTransceiverDirectionSendonly,
				// })
				// if err != nil {
				// 	log.Fatal(fmt.Errorf("main: creating transceiver failed: %w", err))
				// }
				// activeRtpSender = transceiver.Sender()

				activeRtpSender, err = pc.AddTrack(activeVipc.videoTrack)
				if err != nil {
					log.Fatal(fmt.Errorf("main: creating track failed: %w", err))
				}

				_, err := pc.AddTransceiverFromTrack(activeVipc.videoTrack, webrtc.RtpTransceiverInit{
					Direction: webrtc.RTPTransceiverDirectionSendonly,
				})
				if err != nil {
					log.Fatal(fmt.Errorf("main: creating transceiver failed: %w", err))
				}
				// activeRtpSender = transceiver.Sender()
				// activeRtpSender = transceiver.Sender()

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

	decoder, err := NewVipcDecoder()
	if err != nil {
		log.Fatalln(err)
	}
	defer decoder.Free()

	encoder, err := NewVipcEncoder(decoder)
	if err != nil {
		log.Fatalln(err)
	}
	defer encoder.Free()

	for i, c := range VisionIpcCameras {

		visionTrack, err := NewVisionIpcTrack(c, decoder, encoder)
		if err != nil {
			log.Fatal(fmt.Errorf("main: creating track failed: %w", err))
		}

		tracks[i] = visionTrack
	}

	go ActivateStream(0)

	// Handle ffmpeg logs
	astiav.SetLogLevel(astiav.LogLevelError)
	astiav.SetLogCallback(func(l astiav.LogLevel, fmt, msg, parent string) {
		log.Printf("ffmpeg log: %s (level: %d)\n", strings.TrimSpace(msg), l)
	})

	for {
		log.Println("waitng for streaming sig")
		v := <-Streaming
		for camIndex == wantCamIndex {
			sent, err := v.Drain()
			if err != nil {
				log.Println(fmt.Errorf("main: vipc error: %w", err))
				continue
			}
			if sent < 1 {
				continue
			}
		}
	}
}
