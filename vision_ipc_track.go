package main

import (
	"errors"
	"fmt"
	"log"
	"time"

	"github.com/asticode/go-astiav"
	"github.com/commaai/cereal"
	zmq "github.com/pebbe/zmq4"
	"github.com/pion/webrtc/v3"
	"github.com/pion/webrtc/v3/pkg/media"
)

const V4L2_BUF_FLAG_KEYFRAME = uint32(8)

type VisionIpcTrack struct {
	cam            VisionIpcCamera
	name           string
	context        *zmq.Context
	subscriber     *zmq.Socket
	decoder        *astiav.CodecContext
	encoder        *astiav.CodecContext
	decPkt         *astiav.Packet
	decFrame       *astiav.Frame
	encPkt         *astiav.Packet
	lastIdx        int64
	seenIframe     bool
	videoTrack     *webrtc.TrackLocalStaticSample
	networkLatency float64
	frameLatency   float64
	processLatency float64
	pcLatency      float64
	timeQ          []int64
	msgCount       int
	encodeId       uint32
	logMonoTime    uint64
	timestampEof   uint64
	dataSize       int
	counter        int64
	duration       time.Duration
}

type Frame struct {
	Frame *astiav.Frame
	Roll  string
}

func NewVisionIpcTrack(cam VisionIpcCamera, decoder *astiav.CodecContext, encoder *astiav.CodecContext) (v *VisionIpcTrack, err error) {
	v = &VisionIpcTrack{
		cam:            cam,
		name:           cam.prefix + "EncodeData",
		decPkt:         astiav.AllocPacket(),
		encPkt:         astiav.AllocPacket(),
		decFrame:       astiav.AllocFrame(),
		decoder:        decoder,
		encoder:        encoder,
		lastIdx:        -1,
		seenIframe:     false,
		networkLatency: 0.0,
		frameLatency:   0.0,
		processLatency: 0.0,
		pcLatency:      0.0,
		timeQ:          []int64{},
		msgCount:       0,
		encodeId:       0,
		logMonoTime:    0,
		timestampEof:   0,
		dataSize:       0,
		counter:        0,
		duration:       50 * time.Millisecond,
	}

	v.context, err = zmq.NewContext()
	if err != nil {
		log.Fatal(fmt.Errorf("main: new zmq context failed: %w", err))
	}

	v.videoTrack, err = webrtc.NewTrackLocalStaticSample(webrtc.RTPCodecCapability{
		MimeType: webrtc.MimeTypeH264,
	}, "video", "pion")
	if err != nil {
		log.Fatal(fmt.Errorf("vision_ipc_track: creating track failed: %w", err))
	}

	v.subscriber, err = v.newSocket()
	if err != nil {
		log.Fatal(fmt.Errorf("main: zmq connect failed: %w", err))
	}

	return v, nil
}

func (v *VisionIpcTrack) newSocket() (socket *zmq.Socket, err error) {
	socket, err = v.context.NewSocket(zmq.SUB)
	if err != nil {
		return nil, err
	}
	err = socket.Connect(GetServiceURI(v.name))
	if err != nil {
		return nil, err
	}
	return socket, nil
}

func (v *VisionIpcTrack) Subscribe() (err error) {
	return v.subscriber.SetSubscribe("")
}

func (v *VisionIpcTrack) Unsubscribe() (err error) {
	newSub, err := v.newSocket()
	if err != nil {
		return err
	}
	oldSub := v.subscriber
	v.subscriber = newSub
	return oldSub.Close()
}

func (v *VisionIpcTrack) Stop() {
	v.decFrame.Free()
	v.encPkt.Free()
	v.decPkt.Free()
	v.subscriber.Close()
	v.context.Term()
}

func (v *VisionIpcTrack) Drain() (sent int, err error) {
	sent = 0
	msgs := DrainSock(v.subscriber, false)
	v.msgCount = len(msgs)
	if v.msgCount > 0 {
		for _, msg := range msgs {
			evt, err := cereal.ReadRootEvent(msg)
			if err != nil {
				return sent, fmt.Errorf("cereal read root event failed: %w", err)
			}

			var encodeData cereal.EncodeData
			switch v.cam {
			case ROAD:
				encodeData, err = evt.RoadEncodeData()
			case WIDE_ROAD:
				encodeData, err = evt.WideRoadEncodeData()
			case DRIVER:
				encodeData, err = evt.DriverEncodeData()
			}

			if err != nil {
				return sent, fmt.Errorf("cereal get encode data failed: %w", err)
			}

			v.encPkt.Unref()
			v.decPkt.Unref()
			v.decFrame.Unref()

			pts := int64(v.counter * 50) // 50ms per frame

			encodeIndex, err := encodeData.Idx()
			if err != nil {
				return sent, fmt.Errorf("cereal read encode index failed: %w", err)
			}
			v.encodeId = encodeIndex.EncodeId()

			idxFlags := encodeIndex.Flags()

			if v.encodeId != 0 && v.encodeId != uint32(v.lastIdx+1) {
				v.lastIdx = int64(v.encodeId)
				return sent, errors.New("drop packet")
			}

			v.lastIdx = int64(v.encodeId)

			if !v.seenIframe && (idxFlags&V4L2_BUF_FLAG_KEYFRAME) == 0 {
				return sent, errors.New("waiting for iframe")
			}

			v.timeQ = append(v.timeQ, (time.Now().UnixNano() / 1e6))

			v.timestampEof = encodeIndex.TimestampEof()
			timestampSof := encodeIndex.TimestampSof()
			v.logMonoTime = evt.LogMonoTime()

			ts := int64(encodeData.UnixTimestampNanos())
			v.networkLatency = float64(time.Now().UnixNano()-ts) / 1e6
			v.frameLatency = ((float64(v.timestampEof) / 1e9) - (float64(timestampSof) / 1e9)) * 1000
			v.processLatency = ((float64(v.logMonoTime) / 1e9) - (float64(v.timestampEof) / 1e9)) * 1000

			if !v.seenIframe {
				header, err := encodeData.Header()
				if err != nil {
					return sent, fmt.Errorf("cereal read encode header failed: %w", err)
				}

				if err := v.decPkt.FromData(header); err != nil {
					return sent, fmt.Errorf("vision_ipc_track: packet header load failed: %w", err)
				}

				if err := v.decoder.SendPacket(v.decPkt); err != nil {
					return sent, fmt.Errorf("vision_ipc_track: sending packet failed: %w", err)
				}

				v.seenIframe = true
			}

			data, err := encodeData.Data()
			if err != nil {
				return sent, fmt.Errorf("cereal read encode data failed: %w", err)
			}
			v.dataSize = len(data)

			v.decPkt.SetPts(pts)

			if err := v.decPkt.FromData(data); err != nil {
				return sent, fmt.Errorf("vision_ipc_track decoder: packet data load failed: %w", err)
			}

			if err := v.decoder.SendPacket(v.decPkt); err != nil {
				return sent, fmt.Errorf("vision_ipc_track decoder: sending packet failed: %w", err)
			}

			if err := v.decoder.ReceiveFrame(v.decFrame); err != nil {
				return sent, fmt.Errorf("vision_ipc_track decoder: receiving frame failed: %w", err)
			}

			if err = v.encoder.SendFrame(v.decFrame); err != nil {
				return sent, fmt.Errorf("vision_ipc_track encoder: sending frame failed: %w", err)
			}

			if err := v.encoder.ReceivePacket(v.encPkt); err != nil {
				if astiav.ErrEagain.Is(err) {
					// Encoder might need a few frames on startup to get started. Keep going
				} else {
					return sent, fmt.Errorf("vision_ipc_track encoder: receiving frame failed: %w", err)
				}
			}

			if err != nil {
				return sent, fmt.Errorf("vision_ipc_track: encode error: %w", err)
			}
			fmt.Println(len(v.encPkt.Data()))
			err = v.videoTrack.WriteSample(media.Sample{Data: v.encPkt.Data(), Duration: v.duration})
			if err != nil {
				return sent, fmt.Errorf("vision_ipc_track: sample write error: %w", err)
			}

			v.counter++
			sent += 1

			v.pcLatency = ((float64(time.Now().UnixNano()) / 1e6) - float64(v.timeQ[0]))
			v.timeQ = v.timeQ[1:]

			fmt.Printf("%2d %4d %.3f %.3f roll %6.2f ms latency %6.2f ms + %6.2f ms + %6.2f ms = %6.2f ms %d %s\n",
				v.msgCount,
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
	return sent, nil
}
