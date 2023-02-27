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

var cams = []string{
	"roadEncodeData",
	"wideRoadEncodeData",
	"driverEncodeData",
}

type VisionIpcTrackTranscodeStream struct {
	decCodecID      astiav.CodecID
	decCodec        *astiav.Codec
	decCodecContext *astiav.CodecContext
	encCodecID      astiav.CodecID
	encCodec        *astiav.Codec
	encCodecContext *astiav.CodecContext
	decPkt          *astiav.Packet
	decFrame        *astiav.Frame
	encPkt          *astiav.Packet
}

type VisionIpcTrack struct {
	name       string
	stream     *VisionIpcTrackTranscodeStream
	camIdx     int
	context    *zmq.Context
	subscriber *zmq.Socket
	lastIdx    int64
	seenIframe bool

	videoTrack *webrtc.TrackLocalStaticSample

	networkLatency float64
	frameLatency   float64
	processLatency float64
	pcLatency      float64
	timeQ          []int64

	msgCount     int
	encodeId     uint32
	logMonoTime  uint64
	timestampEof uint64
	dataSize     int

	counter int
}

type Frame struct {
	Frame *astiav.Frame
	Roll  string
}

var Open = false

func NewVisionIpcTrack() (track *VisionIpcTrack, err error) {
	// Create a new context
	context, err := zmq.NewContext()
	if err != nil {
		log.Fatal(fmt.Errorf("vision_ipc_track: new zmq context failed: %w", err))
		return nil, err
	}

	// Create stream
	s := &VisionIpcTrackTranscodeStream{}

	s.decCodecID = astiav.CodecIDHevc

	// Find decoder
	if s.decCodec = astiav.FindDecoder(s.decCodecID); s.decCodec == nil {
		return nil, errors.New("vision_ipc_track: codec is nil")
	}

	// Alloc codec context
	if s.decCodecContext = astiav.AllocCodecContext(s.decCodec); s.decCodecContext == nil {
		return nil, errors.New("vision_ipc_track: codec context is nil")
	}

	s.decCodecContext.SetHeight(track.Height())
	s.decCodecContext.SetPixelFormat(track.PixelFormat())
	s.decCodecContext.SetSampleAspectRatio(track.AspectRatio())
	s.decCodecContext.SetTimeBase(track.TimeBase())
	s.decCodecContext.SetWidth(track.Width())

	// Open codec context
	if err := s.decCodecContext.Open(s.decCodec, nil); err != nil {
		log.Fatal(fmt.Errorf("vision_ipc_track: opening codec context failed: %w", err))
		return nil, err
	}

	// Alloc packet
	s.decPkt = astiav.AllocPacket()

	// Alloc frame
	s.decFrame = astiav.AllocFrame()

	//=========
	// ENCODER
	//=========

	// Get codec id
	s.encCodecID = astiav.CodecIDH264

	// Find encoder
	if s.encCodec = astiav.FindEncoder(s.encCodecID); s.encCodec == nil {
		err = errors.New("vision_ipc_track: codec is nil")
		return
	}

	// Alloc codec context
	if s.encCodecContext = astiav.AllocCodecContext(s.encCodec); s.encCodecContext == nil {
		err = errors.New("main: codec context is nil")
		return
	}

	s.encCodecContext.SetHeight(s.decCodecContext.Height())
	s.encCodecContext.SetFramerate(s.decCodecContext.Framerate())
	s.encCodecContext.SetPixelFormat(s.decCodecContext.PixelFormat())
	s.encCodecContext.SetBitRate(1_500_000)
	// s.encCodecContext.SetGopSize(0)
	s.encCodecContext.SetSampleAspectRatio(s.decCodecContext.SampleAspectRatio())
	s.encCodecContext.SetTimeBase(s.decCodecContext.TimeBase())
	s.encCodecContext.SetWidth(s.decCodecContext.Width())
	s.encCodecContext.SetStrictStdCompliance(astiav.StrictStdComplianceExperimental)

	// Update flags
	flags := s.encCodecContext.Flags()
	flags = flags.Add(astiav.CodecContextFlagLowDelay)
	s.encCodecContext.SetFlags(flags)

	// TOOD possibly PR to go-astiav with this SetOpt function
	// av_opt_set(mCodecContext->priv_data, "preset", "ultrafast", 0);
	// av_opt_set(mCodecContext->priv_data, "tune", "zerolatency", 0);
	s.encCodecContext.SetOpt("preset", "ultrafast", 0)
	s.encCodecContext.SetOpt("tune", "zerolatency", 0)
	// from https://trac.ffmpeg.org/ticket/3354
	// interesting: https://github.com/giorgisio/goav/blob/master/avcodec/context.go#L190

	// Open codec context
	if err = s.encCodecContext.Open(s.encCodec, nil); err != nil {
		err = fmt.Errorf("encoder: opening codec context failed: %w", err)
		return
	}

	// Allocate packet
	s.encPkt = astiav.AllocPacket()

	videoTrack, err := webrtc.NewTrackLocalStaticSample(webrtc.RTPCodecCapability{
		MimeType: webrtc.MimeTypeH264,
	}, "video", "pion")
	if err != nil {
		log.Fatal(fmt.Errorf("vision_ipc_track: creating track failed: %w", err))
	}

	return &VisionIpcTrack{
		camIdx:         0,
		stream:         s,
		context:        context,
		lastIdx:        -1,
		seenIframe:     false,
		networkLatency: 0.0,
		frameLatency:   0.0,
		processLatency: 0.0,
		pcLatency:      0.0,
		timeQ:          []int64{},

		videoTrack: videoTrack,

		msgCount:     0,
		encodeId:     0,
		logMonoTime:  0,
		timestampEof: 0,
		dataSize:     0,

		counter: 0,
	}, nil
}

func (v *VisionIpcTrack) TimeBase() (t astiav.Rational) {
	return astiav.NewRational(1, v.FrameRate())
}

func (v *VisionIpcTrack) FrameRate() int {
	return 20
}

func (v *VisionIpcTrack) Height() int {
	return 1208
}

func (v *VisionIpcTrack) Width() int {
	return 1928
}

func (v *VisionIpcTrack) AspectRatio() (t astiav.Rational) {
	return astiav.NewRational(151, 241)
}

func (v *VisionIpcTrack) PixelFormat() (f astiav.PixelFormat) {
	return astiav.PixelFormatYuv420P
}

func (v *VisionIpcTrack) Stop() {
	if !Open {
		return
	}
	Open = false
	v.stream.decFrame.Free()
	v.stream.encPkt.Free()
	v.stream.decPkt.Free()
	v.stream.encCodecContext.Free()
	v.stream.decCodecContext.Free()
	v.context.Term()
}

func (v *VisionIpcTrack) Start() {
	if Open {
		return
	}

	v.Subscribe()
	defer v.subscriber.Close()

	v.counter = 0

	duration := 50 * time.Millisecond

	Open = true
	for Open {
		// if v.subscriber == nil {
		// 	continue
		// }
		msgs := DrainSock(v.subscriber, false)
		v.msgCount = len(msgs)
		if v.msgCount > 0 {
			for _, msg := range msgs {

				v.stream.encPkt.Unref()
				v.stream.decPkt.Unref()
				v.stream.decFrame.Unref()

				pts := int64(v.counter * 50) // 50ms per frame

				evt, err := cereal.ReadRootEvent(msg)
				if err != nil {
					log.Println(fmt.Errorf("cereal read root event failed: %w", err))
					continue
				}

				var encodeData cereal.EncodeData
				switch evt.Which() {
				case cereal.Event_Which_roadEncodeData:
					encodeData, err = evt.RoadEncodeData()
				case cereal.Event_Which_wideRoadEncodeData:
					encodeData, err = evt.WideRoadEncodeData()
				case cereal.Event_Which_driverEncodeData:
					encodeData, err = evt.DriverEncodeData()
				}
				if err != nil {
					log.Println(fmt.Errorf("cereal read road encode data failed: %w", err))
					continue
				}

				encodeIndex, err := encodeData.Idx()
				if err != nil {
					log.Println(fmt.Errorf("cereal read encode index failed: %w", err))
					continue
				}

				v.encodeId = encodeIndex.EncodeId()

				idxFlags := encodeIndex.Flags()

				if v.encodeId != 0 && v.encodeId != uint32(v.lastIdx+1) {
					fmt.Println("DROP PACKET!")
				}

				v.lastIdx = int64(v.encodeId)

				if !v.seenIframe && (idxFlags&V4L2_BUF_FLAG_KEYFRAME) == 0 {
					fmt.Println("waiting for iframe")
					continue
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
					// Decode video frame

					// AvPacketFromData
					header, err := encodeData.Header()
					if err != nil {
						log.Println(fmt.Errorf("cereal read encode header failed: %w", err))
						continue
					}

					if err := v.stream.decPkt.FromData(header); err != nil {
						log.Println(fmt.Errorf("vision_ipc_track: packet header load failed: %w", err))
						continue
					}

					// Send packet
					if err := v.stream.decCodecContext.SendPacket(v.stream.decPkt); err != nil {
						log.Println(fmt.Errorf("vision_ipc_track: sending packet failed: %w", err))
						continue
					}

					v.seenIframe = true
				}

				// AvPacketFromData
				data, err := encodeData.Data()
				if err != nil {
					log.Println(fmt.Errorf("cereal read encode data failed: %w", err))
					continue
				}
				v.dataSize = len(data)

				v.stream.decPkt.SetPts(pts)

				if err := v.stream.decPkt.FromData(data); err != nil {
					log.Println(fmt.Errorf("vision_ipc_track decoder: packet data load failed: %w", err))
					continue
				}

				// Send packet
				if err := v.stream.decCodecContext.SendPacket(v.stream.decPkt); err != nil {
					log.Println(fmt.Errorf("vision_ipc_track decoder: sending packet failed: %w", err))
					continue
				}

				if err := v.stream.decCodecContext.ReceiveFrame(v.stream.decFrame); err != nil {
					log.Println(fmt.Errorf("vision_ipc_track decoder: receiving frame failed: %w", err))
					continue
				}

				v.pcLatency = ((float64(time.Now().UnixNano()) / 1e6) - float64(v.timeQ[0]))
				v.timeQ = v.timeQ[1:]

				// this didn't work
				// // if counter is 0, we changed camera source, so encode a keyframe!
				// if v.counter == 0 {
				// 	// v.stream.encPkt.SetFlags(v.stream.encPkt.Flags().Add(astiav.PacketFlagKey))
				// 	// av_opt_set(codec_ctx->priv_data, "x264opts", "keyint=1", 0);
				// 	// v.stream.encCodecContext.SetOpt("keyint")
				// 	// v.stream.encCodecContext.SetGopSize(1)
				// 	v.stream.encCodecContext.SetOpt("x264opts", "keyint", 1)
				// }

				// Send frame
				if err = v.stream.encCodecContext.SendFrame(v.stream.decFrame); err != nil {
					log.Println(fmt.Errorf("vision_ipc_track encoder: sending frame failed: %w", err))
					continue
				}

				if err := v.stream.encCodecContext.ReceivePacket(v.stream.encPkt); err != nil {
					if astiav.ErrEagain.Is(err) {
						// Encoder might need a few frames on startup to get started. Keep going
					} else {
						log.Println(fmt.Errorf("vision_ipc_track encoder: receiving frame failed: %w", err))
					}
					continue
				}

				// this didn't work
				// // if counter is 0, we changed camera source, so encode a keyframe!
				// if v.counter == 0 {
				// 	// v.stream.encPkt.SetFlags(v.stream.encPkt.Flags().Add(astiav.PacketFlagKey))
				// 	// av_opt_set(codec_ctx->priv_data, "x264opts", "keyint=1", 0);
				// 	// v.stream.encCodecContext.SetOpt("keyint")
				// 	// v.stream.encCodecContext.SetGopSize(0)
				// 	v.stream.encCodecContext.SetOpt("x264opts", "keyint", 0)
				// }

				if err != nil {
					log.Println(fmt.Errorf("vision_ipc_track: encode error: %w", err))
					continue
				}

				if err = v.videoTrack.WriteSample(media.Sample{Data: v.stream.encPkt.Data(), Duration: duration}); err != nil {
					log.Println(fmt.Errorf("vision_ipc_track: sample write error: %w", err))
					continue
				}

				v.counter++

				v.PrintStats()
			}
		}
	}
}

func (v *VisionIpcTrack) PrintStats() {
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

func (v *VisionIpcTrack) ChangeCamera() {
	v.camIdx = (v.camIdx + 1) % len(cams)
	v.Subscribe()
}

func (v *VisionIpcTrack) NewSocket() (socket *zmq.Socket, err error) {
	socket, err = v.context.NewSocket(zmq.SUB)
	if err != nil {
		return nil, err
	}
	v.name = cams[v.camIdx]
	err = socket.Connect(GetServiceURI(v.name))
	if err != nil {
		return nil, err
	}
	return socket, nil
}

func (v *VisionIpcTrack) Subscribe() {
	oldSub := v.subscriber
	subscriber, err := v.NewSocket()
	if err != nil {
		log.Println(fmt.Errorf("zmq subscribe failed: %w", err))
	}
	subscriber.SetSubscribe("")
	v.counter = 0
	v.subscriber = subscriber
	if oldSub != nil {
		oldSub.Close()
	}
}
