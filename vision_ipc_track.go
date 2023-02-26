package main

import (
	"fmt"
	"log"
	"time"

	"github.com/asticode/go-astiav"
	"github.com/commaai/cereal"
	zmq "github.com/pebbe/zmq4"
	"github.com/pion/webrtc/v3"
	"github.com/pion/webrtc/v3/pkg/media"
)

type VisionIpcCamera struct {
	prefix        string
	getEncodeData func(s cereal.Event) (cereal.EncodeData, error)
}

var (
	ROAD = VisionIpcCamera{prefix: "road",
		getEncodeData: func(s cereal.Event) (cereal.EncodeData, error) {
			return s.RoadEncodeData()
		},
	}
	WIDE_ROAD = VisionIpcCamera{prefix: "wideRoad",
		getEncodeData: func(s cereal.Event) (cereal.EncodeData, error) {
			return s.WideRoadEncodeData()
		},
	}
	DRIVER = VisionIpcCamera{prefix: "driver",
		getEncodeData: func(s cereal.Event) (cereal.EncodeData, error) {
			return s.DriverEncodeData()
		},
	}
)

var VisionIpcCameras = []VisionIpcCamera{ROAD, WIDE_ROAD, DRIVER}

const V4L2_BUF_FLAG_KEYFRAME = uint32(8)

type VisionIpcTrack struct {
	cam        VisionIpcCamera
	name       string
	stream     *VIPCTranscoderStream
	transcoder *VIPCTranscoder
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

	counter  int64
	duration time.Duration
}

type Frame struct {
	Frame *astiav.Frame
	Roll  string
}

var Open = false

func NewVisionIpcTrack(cam VisionIpcCamera, transcoder VIPCTranscoder) (v *VisionIpcTrack, err error) {
	v = &VisionIpcTrack{
		cam:            cam,
		name:           cam.prefix + "EncodeData",
		stream:         &VIPCTranscoderStream{},
		transcoder:     &transcoder,
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

	// Alloc packet
	v.stream.decPkt = astiav.AllocPacket()

	// Alloc frame
	v.stream.decFrame = astiav.AllocFrame()

	// Allocate packet
	v.stream.encPkt = astiav.AllocPacket()

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
	if !Open {
		return
	}
	Open = false
	v.stream.decFrame.Free()
	v.stream.encPkt.Free()
	v.stream.decPkt.Free()
	v.subscriber.Close()
	v.context.Term()
}

func (v *VisionIpcTrack) ProcessCerealEvent(evt cereal.Event) {
	v.stream.encPkt.Unref()
	v.stream.decPkt.Unref()
	v.stream.decFrame.Unref()

	pts := int64(v.counter * 50) // 50ms per frame

	encodeData, err := v.cam.getEncodeData(evt)
	if err != nil {
		log.Println(fmt.Errorf("cereal read road encode data failed: %w", err))
		return
	}

	encodeIndex, err := encodeData.Idx()
	if err != nil {
		log.Println(fmt.Errorf("cereal read encode index failed: %w", err))
		return
	}

	v.encodeId = encodeIndex.EncodeId()

	idxFlags := encodeIndex.Flags()

	if v.encodeId != 0 && v.encodeId != uint32(v.lastIdx+1) {
		fmt.Println("DROP PACKET!")
	}

	v.lastIdx = int64(v.encodeId)

	if !v.seenIframe && (idxFlags&V4L2_BUF_FLAG_KEYFRAME) == 0 {
		fmt.Println("waiting for iframe")
		return
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
			log.Println(fmt.Errorf("cereal read encode header failed: %w", err))
			return
		}

		if err := v.stream.decPkt.FromData(header); err != nil {
			log.Println(fmt.Errorf("vision_ipc_track: packet header load failed: %w", err))
			return
		}

		if err := v.transcoder.decoder.decCodecContext.SendPacket(v.stream.decPkt); err != nil {
			log.Println(fmt.Errorf("vision_ipc_track: sending packet failed: %w", err))
			return
		}

		v.seenIframe = true
	}

	data, err := encodeData.Data()
	if err != nil {
		log.Println(fmt.Errorf("cereal read encode data failed: %w", err))
		return
	}
	v.dataSize = len(data)

	v.stream.decPkt.SetPts(pts)

	if err := v.stream.decPkt.FromData(data); err != nil {
		log.Println(fmt.Errorf("vision_ipc_track decoder: packet data load failed: %w", err))
		return
	}

	if err := v.transcoder.decoder.decCodecContext.SendPacket(v.stream.decPkt); err != nil {
		log.Println(fmt.Errorf("vision_ipc_track decoder: sending packet failed: %w", err))
		return
	}

	if err := v.transcoder.decoder.decCodecContext.ReceiveFrame(v.stream.decFrame); err != nil {
		log.Println(fmt.Errorf("vision_ipc_track decoder: receiving frame failed: %w", err))
		return
	}

	if err = v.transcoder.encoder.encCodecContext.SendFrame(v.stream.decFrame); err != nil {
		log.Println(fmt.Errorf("vision_ipc_track encoder: sending frame failed: %w", err))
		return
	}

	if err := v.transcoder.encoder.encCodecContext.ReceivePacket(v.stream.encPkt); err != nil {
		if astiav.ErrEagain.Is(err) {
			// Encoder might need a few frames on startup to get started. Keep going
		} else {
			log.Println(fmt.Errorf("vision_ipc_track encoder: receiving frame failed: %w", err))
		}
		return
	}

	if err != nil {
		log.Println(fmt.Errorf("vision_ipc_track: encode error: %w", err))
		return
	}

	if err = v.videoTrack.WriteSample(media.Sample{Data: v.stream.encPkt.Data(), Duration: v.duration}); err != nil {
		log.Println(fmt.Errorf("vision_ipc_track: sample write error: %w", err))
		return
	}

	v.counter++

	v.pcLatency = ((float64(time.Now().UnixNano()) / 1e6) - float64(v.timeQ[0]))
	v.timeQ = v.timeQ[1:]
}

func (v *VisionIpcTrack) AddTrack(peerConnection *webrtc.PeerConnection) *webrtc.RTPSender {
	rtpSender, err := peerConnection.AddTrack(v.videoTrack)
	if err != nil {
		log.Fatal(fmt.Errorf("main: creating track failed: %w", err))
	}

	_, err = peerConnection.AddTransceiverFromTrack(v.videoTrack,
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

	return rtpSender
}
