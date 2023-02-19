package main

import (
	"fmt"

	"capnproto.org/go/capnp/v3"
	"github.com/commaai/cereal"
	zmq "github.com/pebbe/zmq4"
)

func GetServicePort(name string) int {
	port := -1
	for _, s := range cereal.GetServices() {
		if s.Name == name {
			port = s.Port
			break
		}
	}

	if port < 0 {
		panic("invalid endpoint")
	}

	return port
}

func DrainSock(socket *zmq.Socket, waitForOne bool) []*capnp.Message {
	var ret []*capnp.Message
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
		msg, err := capnp.Unmarshal(dat)
		if err != nil {
			panic(err)
		}
		ret = append(ret, msg)
	}
	return ret
}

func GetServiceURI(name string) string {
	port := GetServicePort(name)
	return fmt.Sprintf("tcp://tici:%d", port)
}
