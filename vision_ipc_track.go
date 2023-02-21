package main

import (
	"errors"
	"fmt"
	"log"
	"time"

	"github.com/asticode/go-astiav"
	"github.com/commaai/cereal"
	zmq "github.com/pebbe/zmq4"
)

const V4L2_BUF_FLAG_KEYFRAME = uint32(8)

type Stream struct {
	decCodec        *astiav.Codec
	decCodecContext *astiav.CodecContext
	inputStream     *astiav.Stream
}

type VisionIpcTrack struct {
	name       string
	stream     *Stream
	context    *zmq.Context
	subscriber *zmq.Socket
	lastIdx    int64
	seenIframe bool
	pkt        *astiav.Packet
	f          *astiav.Frame

	networkLatency float64
	frameLatency   float64
	processLatency float64
	pcLatency      float64
	timeQ          []int64
}

func NewVisionIpcTrack(name string) (track *VisionIpcTrack, err error) {
	// Create a new context
	context, err := zmq.NewContext()
	if err != nil {
		log.Fatal(fmt.Errorf("main: new zmq context failed: %w", err))
		return nil, err
	}

	// Connect to the socket
	subscriber, err := context.NewSocket(zmq.SUB)
	if err != nil {
		log.Fatal(fmt.Errorf("main: new zmq socket failed: %w", err))
		return nil, err
	}
	subscriber.SetSubscribe("")
	subscriber.Connect(GetServiceURI(name))

	// Create stream
	s := &Stream{inputStream: nil}

	// Find decoder
	if s.decCodec = astiav.FindDecoder(astiav.CodecIDHevc); s.decCodec == nil {
		return nil, errors.New("main: codec is nil")
	}

	// Alloc codec context
	if s.decCodecContext = astiav.AllocCodecContext(s.decCodec); s.decCodecContext == nil {
		return nil, errors.New("main: codec context is nil")
	}

	// Open codec context
	if err := s.decCodecContext.Open(s.decCodec, nil); err != nil {
		log.Fatal(fmt.Errorf("main: opening codec context failed: %w", err))
		return nil, err
	}

	// Alloc packet
	pkt := astiav.AllocPacket()

	// Alloc frame
	f := astiav.AllocFrame()

	return &VisionIpcTrack{
		name:           name,
		stream:         s,
		context:        context,
		subscriber:     subscriber,
		lastIdx:        -1,
		seenIframe:     false,
		f:              f,
		pkt:            pkt,
		networkLatency: 0.0,
		frameLatency:   0.0,
		processLatency: 0.0,
		pcLatency:      0.0,
		timeQ:          []int64{},
	}, nil
}

func (v *VisionIpcTrack) Close() {
	v.f.Free()
	v.pkt.Free()
	v.stream.decCodecContext.Free()
	v.subscriber.Close()
	v.context.Term()
}

func (v *VisionIpcTrack) ReceiveFrame() (frame *astiav.Frame, err error) {
	for {

		msgs := DrainSock(v.subscriber, true)
		if len(msgs) > 0 {
			for _, msg := range msgs {

				evt, err := cereal.ReadRootEvent(msg)
				if err != nil {
					log.Fatal(fmt.Errorf("cereal read root event failed: %w", err))
					return nil, err
				}

				encodeData, err := evt.RoadEncodeData()
				if err != nil {
					log.Fatal(fmt.Errorf("cereal read road encode data failed: %w", err))
					return nil, err
				}

				encodeIndex, err := encodeData.Idx()
				if err != nil {
					log.Fatal(fmt.Errorf("cereal read encode index failed: %w", err))
					return nil, err
				}

				encodeId := encodeIndex.EncodeId()
				idxFlags := encodeIndex.Flags()

				if encodeId != 0 && encodeId != uint32(v.lastIdx+1) {
					fmt.Println("DROP PACKET!")
				}

				v.lastIdx = int64(encodeId)

				if !v.seenIframe && (idxFlags&V4L2_BUF_FLAG_KEYFRAME) == 0 {
					fmt.Println("waiting for iframe")
					continue
				}
				v.timeQ = append(v.timeQ, (time.Now().UnixNano() / 1e6))

				timestampEof := encodeIndex.TimestampEof()
				timestampSof := encodeIndex.TimestampSof()

				logMonoTime := evt.LogMonoTime()

				ts := int64(encodeData.UnixTimestampNanos())
				v.networkLatency = float64(time.Now().UnixNano()-ts) / 1e6

				v.frameLatency = ((float64(timestampEof) / 1e9) - (float64(timestampSof) / 1e9)) * 1000
				v.processLatency = ((float64(logMonoTime) / 1e9) - (float64(timestampEof) / 1e9)) * 1000

				if !v.seenIframe {
					// Decode video frame

					// AvPacketFromData
					header, err := encodeData.Header()
					if err != nil {
						log.Fatal(fmt.Errorf("cereal read encode header failed: %w", err))
						return nil, err
					}

					if err := v.pkt.FromData(header); err != nil {
						log.Fatal(fmt.Errorf("main: packet header load failed: %w", err))
						return nil, err
					}

					// Send packet
					if err := v.stream.decCodecContext.SendPacket(v.pkt); err != nil {
						log.Fatal(fmt.Errorf("main: sending packet failed: %w", err))
						return nil, err
					}

					v.seenIframe = true
				}

				// AvPacketFromData
				data, err := encodeData.Data()
				if err != nil {
					log.Fatal(fmt.Errorf("cereal read encode data failed: %w", err))
					return nil, err
				}

				if err := v.pkt.FromData(data); err != nil {
					log.Fatal(fmt.Errorf("main: packet header load failed: %w", err))
					return nil, err
				}

				// Send packet
				if err := v.stream.decCodecContext.SendPacket(v.pkt); err != nil {
					log.Fatal(fmt.Errorf("main: sending packet failed: %w", err))
					return nil, err
				}

				if err := v.stream.decCodecContext.ReceiveFrame(v.f); err != nil {
					log.Fatal(fmt.Errorf("main: receiving frame failed: %w", err))
					return nil, err
				}

				v.pcLatency = ((float64(time.Now().UnixNano()) / 1e6) - float64(v.timeQ[0]))
				v.timeQ = v.timeQ[1:]

				fmt.Printf("%2d %4d %.3f %.3f roll %6.2f ms latency %6.2f ms + %6.2f ms + %6.2f ms = %6.2f ms %d %s\n",
					len(msgs),
					encodeId,
					(float64(logMonoTime) / 1e9),
					(float64(encodeIndex.TimestampEof()) / 1e6),
					v.frameLatency,
					v.processLatency,
					v.networkLatency,
					v.pcLatency,
					v.processLatency+v.networkLatency+v.pcLatency,
					len(data),
					v.name,
				)

				return v.f, nil
			}
		}
	}
}
