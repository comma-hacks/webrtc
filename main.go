package main

import (
	"errors"
	"fmt"
	"io"
	"log"
	"secureput"
	"strings"

	"github.com/asticode/go-astiav"
	"github.com/pion/webrtc/v3"
)

var visionTrack *VisionIpcTrack

func ReplaceTrack(prefix string, peerConnection *webrtc.PeerConnection) {
	var err error
	if visionTrack != nil {
		visionTrack.Stop()
	}
	visionTrack, err = NewVisionIpcTrack(prefix + "EncodeData")
	if err != nil {
		log.Fatal(fmt.Errorf("main: creating track failed: %w", err))
	}
	// Create a video track
	videoTrack, err := webrtc.NewTrackLocalStaticRTP(webrtc.RTPCodecCapability{MimeType: webrtc.MimeTypeH264}, "video", "pion")
	if err != nil {
		panic(err)
	}
	rtpSender, err := peerConnection.AddTrack(videoTrack)
	if err != nil {
		panic(err)
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
			astiav.SetLogLevel(astiav.LogLevelError)
			astiav.SetLogCallback(func(l astiav.LogLevel, fmt, msg, parent string) {
				log.Printf("ffmpeg log: %s (level: %d)\n", strings.TrimSpace(msg), l)
			})
			go func() {
				encoderParams := EncoderParams{
					Width:       visionTrack.Width(),
					Height:      visionTrack.Height(),
					TimeBase:    visionTrack.TimeBase(),
					AspectRatio: visionTrack.AspectRatio(),
					PixelFormat: visionTrack.PixelFormat(),
				}

				encoder, err := NewEncoder(encoderParams)

				if err != nil {
					log.Fatal(fmt.Errorf("main: creating track failed: %w", err))
					return
				}
				defer encoder.Close()

				go visionTrack.Start()
				defer visionTrack.Stop()

				for visionTrack != nil {
					var rtpPacket []byte
					for frame := range visionTrack.Frame {
						// Do something with decoded frame
						fmt.Println(frame.Roll)
						// so right now frame is a raw decoded AVFrame
						// https://github.com/FFmpeg/FFmpeg/blob/n5.0/libavutil/frame.h#L317
						avframe := frame.Frame

						// we need to:
						// 1. transcode to h264 with adaptive bitrate using astiav
						outFrame, err := encoder.Encode(avframe)
						if err != nil {
							log.Println(fmt.Errorf("encode error: %w", err))

							continue
						}

						// 2. create an RTP packet with h264 data inside using pion's rtp packetizer
						fmt.Println(len(outFrame.Data()))

						// 3. write the RTP packet to the videoTrack
						if _, err = videoTrack.Write(rtpPacket); err != nil {
							if errors.Is(err, io.ErrClosedPipe) {
								// The peerConnection has been closed.
								return
							}

							log.Println(fmt.Errorf("rtp write error: %w", err))
						}
					}
				}
			}()
		}
	})
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
	for {
		select {}
	}
}
