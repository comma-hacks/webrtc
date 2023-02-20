module github.com/kfatehi/webrtc-body

go 1.19

require github.com/pebbe/zmq4 v1.2.9

require github.com/giorgisio/goav v0.1.0

require (
	capnproto.org/go/capnp/v3 v3.0.0
	golang.org/x/sync v0.0.0-20201020160332-67f06af15bc9 // indirect
	github.com/commaai/cereal v0.0.0
)


replace "github.com/commaai/cereal" v0.0.0 => "./cereal/gen/go"
replace "capnproto.org/go/capnp/v3" v3.0.0 => "./go-capnproto2"