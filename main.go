package main

import (
	"fmt"
	"os"
	"unsafe"

	"github.com/giorgisio/goav/avcodec"
	"github.com/giorgisio/goav/avutil"

	"github.com/commaai/cereal"
	zmq "github.com/pebbe/zmq4"
)

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
			msgs := DrainSock(subscriber, true)
			if len(msgs) > 0 {
				for _, msg := range msgs {

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
					idxFlags := idx.Flags()
					frameId := idx.FrameId()
					segmentNum := idx.SegmentNum()

					fmt.Printf("ts: %d,frameId: %d,segmentNum: %d,encodeId: %d,idxFlags: %d\n", ts, frameId, segmentNum, encodeId, idxFlags)

					continue
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
						pkt := avcodec.AvPacketAlloc()
						if pkt == nil {
							panic("cannot allocate packet")
						}

						// AvPacketFromData
						header, err := evt.Header()
						if err != nil {
							panic(err)
						}
						headerPtr := unsafe.Pointer(&header[0])
						pkt.AvPacketFromData((*uint8)(headerPtr), len(header))
						// pkt.AvInitPacket()
						pkt.SetFlags(pkt.Flags() | avcodec.AV_CODEC_FLAG_TRUNCATED)

						response := codecContext.AvcodecSendPacket(pkt)
						if response < 0 {
							fmt.Printf("Error while sending a packet to the decoder: %s\n", avutil.ErrorFromCode(response))
							// panic("decoding error")
						}
						if response >= 0 {
							response = codecContext.AvcodecReceiveFrame((*avcodec.Frame)(unsafe.Pointer(pFrame)))
							if response == avutil.AvErrorEAGAIN || response == avutil.AvErrorEOF {
								fmt.Println("decode error")
								break
							} else if response < 0 {
								fmt.Printf("Error while receiving a frame from the decoder: %s\n", avutil.ErrorFromCode(response))
								fmt.Println("DROP SURFACE")
								continue
							}
							fmt.Println("??")
							seenIframe = true
						}
						fmt.Println("sss")

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
