package main

import (
	"fmt"
	"os"
	"unsafe"

	"capnproto.org/go/capnp/v3"
	"github.com/giorgisio/goav/avcodec"
	"github.com/giorgisio/goav/avutil"

	zmq "github.com/pebbe/zmq4"

	"github.com/commaai/cereal"
)

type Service struct {
	name       string
	port       int
	should_log bool
	frequency  int
	decimation int
}

var services = []Service{
	{"roadEncodeData", 8062, false, 20, -1},
	{"driverEncodeData", 8063, false, 20, -1},
	{"wideRoadEncodeData", 8064, false, 20, -1},
}

func getPort(endpoint string) int {
	port := -1
	for _, s := range services {
		if s.name == endpoint {
			port = s.port
			break
		}
	}

	if port < 0 {
		panic("invalid endpoint")
	}

	return port
}

func drainSock(socket *zmq.Socket, waitForOne bool) [][]byte {
	// XXX need to use captproto
	// https://github.com/capnproto/go-capnproto2
	var ret [][]byte
	for {
		var dat []byte
		var err error
		if waitForOne && len(ret) == 0 {
			dat, err = socket.RecvBytes(0)
		} else {
			dat, err = socket.RecvBytes(zmq.DONTWAIT)
		}
		if err != nil {
			break
		}
		ret = append(ret, dat)
	}
	return ret
}

func main() {
	// Create a new context
	context, _ := zmq.NewContext()
	defer context.Term()

	// Connect to the socket
	subscriber, _ := context.NewSocket(zmq.SUB)
	defer subscriber.Close()
	service := "roadEncodeData"
	port := getPort(service)
	out := fmt.Sprintf("tcp://tici:%d", port)
	fmt.Println(out)
	subscriber.SetSubscribe("")
	subscriber.Connect(out)

	avcodec.AvcodecRegisterAll()

	// Find the HEVC decoder
	codec := avcodec.AvcodecFindDecoderByName("hevc")
	if codec == nil {
		fmt.Println("HEVC decoder not found")
		return
	}

	codecContext := codec.AvcodecAllocContext3()
	if codecContext == nil {
		fmt.Println("Failed to allocate codec context")
		os.Exit(1)
	}

	// Open codec
	if codecContext.AvcodecOpen2(codec, nil) < 0 {
		fmt.Println("Could not open codec")
		os.Exit(1)
	}

	var lastIdx = -1
	var seenIframe = false
	var V4L2_BUF_FLAG_KEYFRAME = uint32(8)
	for {
		// Allocate video frame
		pFrame := avutil.AvFrameAlloc()
		var frame []byte
		for frame == nil {
			msgs := drainSock(subscriber, true)
			if len(msgs) > 0 {
				for _, rawMsg := range msgs {
					msg, err := capnp.Unmarshal(rawMsg)
					if err != nil {
						panic(err)
					}

					evt, err := cereal.ReadRootEncodeData(msg)
					if err != nil {
						panic(err)
					}

					ts := evt.UnixTimestampNanos()

					idx, err := evt.Idx()
					if err != nil {
						panic(err)
					}
					encodeId := idx.EncodeId()

					fmt.Printf("ts: %d,encodeId: %d\n", ts, encodeId)

					if encodeId != 0 && encodeId != uint32(lastIdx+1) {
						fmt.Println("DROP PACKET!")
					} else {
						lastIdx = int(encodeId)

						idxFlags := idx.Flags()

						if !seenIframe && (idxFlags&V4L2_BUF_FLAG_KEYFRAME) != 0 {
							fmt.Println("waiting for iframe")
							continue
						}

						if !seenIframe {
							// Decode video frame
							pkt := avcodec.AvPacketAlloc()
							if pkt == nil {
								panic("cannot allocate packet")
							}

							pkt.AvInitPacket()
							pkt.SetFlags(pkt.Flags() | avcodec.AV_CODEC_FLAG_TRUNCATED)

							response := codecContext.AvcodecSendPacket(packet)
							if response < 0 {
								fmt.Printf("Error while sending a packet to the decoder: %s\n", avutil.ErrorFromCode(response))
								panic("decoding error")
							}
							if response >= 0 {
								response = codecContext.AvcodecReceiveFrame((*avcodec.Frame)(unsafe.Pointer(pFrame)))
								if response == avutil.AvErrorEAGAIN || response == avutil.AvErrorEOF {
									break
								} else if response < 0 {
									fmt.Printf("Error while receiving a frame from the decoder: %s\n", avutil.ErrorFromCode(response))
									fmt.Println("DROP SURFACE")
									continue
								}
								seenIframe = true
							}
						}

						// frames, err := codec.Decode(av.Packet{Data: evt.Data()})
						// if err != nil {
						// 	fmt.Println("DROP SURFACE")
						// 	continue
						// }
						// if len(frames) == 0 {
						// 	fmt.Println("DROP SURFACE")
						// 	continue
						// }
						// assert(len(frames) == 1)

						// frame = frames[0].Data
						// framePts := frames[0].Pts
						// frameTimebase := frames[0].Timebase

						// frame = append(append(make([]byte, 0, len(frame)+FrameHeaderLen), encodeUint32(uint32(len(frame)))...), frame...)
						// binary.LittleEndian.PutUint32(frame[4:], encodeUint64(framePts))
						// binary.LittleEndian.PutUint32(frame[12:], encodeUint64(frameTimebase))
					}
				}
			}
		}
		fmt.Println("Received frame with size:", len(frame), "bytes")
	}
}
