package main

import (
	"fmt"

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

	for {
		var frame []byte
		for frame == nil {
			msgs := drainSock(subscriber, true)
			if len(msgs) > 0 {
				frame = msgs[0]
			}
		}
		fmt.Println("Received frame with size:", len(frame), "bytes")
	}
}
