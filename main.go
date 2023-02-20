package main

import (
	"errors"
	"fmt"
	"log"
	"strings"

	"github.com/asticode/go-astiav"

	"github.com/commaai/cereal"
	zmq "github.com/pebbe/zmq4"
)

type stream struct {
	decCodec        *astiav.Codec
	decCodecContext *astiav.CodecContext
	inputStream     *astiav.Stream
}

func main() {
	// Create a new context
	context, _ := zmq.NewContext()
	defer context.Term()

	// Connect to the socket
	subscriber, _ := context.NewSocket(zmq.SUB)
	defer subscriber.Close()

	out := GetServiceURI("roadEncodeData")
	fmt.Println(out)
	subscriber.SetSubscribe("")
	subscriber.Connect(out)

	// Handle ffmpeg logs
	astiav.SetLogLevel(astiav.LogLevelDebug)
	astiav.SetLogCallback(func(l astiav.LogLevel, fmt, msg, parent string) {
		log.Printf("ffmpeg log: %s (level: %d)\n", strings.TrimSpace(msg), l)
	})

	// Create stream
	s := &stream{inputStream: nil}

	// Find decoder
	if s.decCodec = astiav.FindDecoder(astiav.CodecIDHevc); s.decCodec == nil {
		log.Fatal(errors.New("main: codec is nil"))
	}

	// Alloc codec context
	if s.decCodecContext = astiav.AllocCodecContext(s.decCodec); s.decCodecContext == nil {
		log.Fatal(errors.New("main: codec context is nil"))
	}
	defer s.decCodecContext.Free()

	// Open codec context
	if err := s.decCodecContext.Open(s.decCodec, nil); err != nil {
		log.Fatal(fmt.Errorf("main: opening codec context failed: %w", err))
	}

	var lastIdx = -1
	var seenIframe = false
	var V4L2_BUF_FLAG_KEYFRAME = uint32(8)

	// Alloc packet
	pkt := astiav.AllocPacket()
	defer pkt.Free()

	// Alloc frame
	f := astiav.AllocFrame()
	defer f.Free()

	for {
		var frame []byte
		for frame == nil {
			msgs := DrainSock(subscriber, true)
			if len(msgs) > 0 {
				for _, msg := range msgs {

					evt, err := cereal.ReadRootEvent(msg)
					if err != nil {
						panic(err)
					}

					encodeData, err := evt.RoadEncodeData()
					if err != nil {
						panic(err)
					}

					encodeIndex, err := encodeData.Idx()
					if err != nil {
						panic(err)
					}

					encodeId := encodeIndex.EncodeId()
					idxFlags := encodeIndex.Flags()

					if encodeId != 0 && encodeId != uint32(lastIdx+1) {
						fmt.Println("DROP PACKET!")
					}

					lastIdx = int(encodeId)

					if !seenIframe && (idxFlags&V4L2_BUF_FLAG_KEYFRAME) == 0 {
						fmt.Println("waiting for iframe")
						continue
					}

					if !seenIframe {
						// Decode video frame

						// AvPacketFromData
						header, err := encodeData.Header()
						if err != nil {
							panic(err)
						}

						if err := pkt.FromData(header); err != nil {
							log.Fatal(fmt.Errorf("main: packet header load failed: %w", err))
						}

						// Send packet
						if err := s.decCodecContext.SendPacket(pkt); err != nil {
							log.Fatal(fmt.Errorf("main: sending packet failed: %w", err))
						}

						seenIframe = true
					}

					// AvPacketFromData
					data, err := encodeData.Data()
					if err != nil {
						panic(err)
					}

					if err := pkt.FromData(data); err != nil {
						log.Fatal(fmt.Errorf("main: packet header load failed: %w", err))
					}

					// Send packet
					if err := s.decCodecContext.SendPacket(pkt); err != nil {
						log.Fatal(fmt.Errorf("main: sending packet failed: %w", err))
					}

					if err := s.decCodecContext.ReceiveFrame(f); err != nil {
						if errors.Is(err, astiav.ErrEof) || errors.Is(err, astiav.ErrEagain) {
							break
						}
						log.Fatal(fmt.Errorf("main: receiving frame failed: %w", err))
					}

					// Do something with decoded frame
					log.Printf("new frame: stream %d - pts: %d", pkt.StreamIndex(), f.Pts())
				}
			}
		}
	}
}
