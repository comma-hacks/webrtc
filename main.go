package main

import (
	"fmt"
	"os"

	"github.com/giorgisio/goav/avcodec"

	zmq "github.com/pebbe/zmq4"
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

	for {
		var frame []byte
		for frame == nil {
			msgs := drainSock(subscriber, true)
			if len(msgs) > 0 {
				/* Port the following Python aiortc code to Golang for Pion
				   with evt_context as evt:
				       evta = getattr(evt, evt.which())
				       if evta.idx.encodeId != 0 and evta.idx.encodeId != (self.last_idx+1):
				           print("DROP PACKET!")
				       self.last_idx = evta.idx.encodeId
				       if not self.seen_iframe and not (evta.idx.flags & V4L2_BUF_FLAG_KEYFRAME):
				           print("waiting for iframe")
				           continue
				       self.time_q.append(time.monotonic())

				       # put in header (first)
				       if not self.seen_iframe:
				           self.codec.decode(av.packet.Packet(evta.header))
				           self.seen_iframe = True

				       frames = self.codec.decode(av.packet.Packet(evta.data))
				       if len(frames) == 0:
				           print("DROP SURFACE")
				           continue
				       assert len(frames) == 1

				       frame = frames[0]
				       frame.pts = pts
				       frame.time_base = time_base
				*/
			}
		}
		fmt.Println("Received frame with size:", len(frame), "bytes")
	}
}
