module github.com/kfatehi/webrtc-body

go 1.19

require github.com/pebbe/zmq4 v1.2.9

require github.com/stretchr/testify v1.8.1 // indirect

require (
	capnproto.org/go/capnp/v3 v3.0.0
	github.com/commaai/cereal v0.0.0
	golang.org/x/sync v0.0.0-20220722155255-886fb9371eb4 // indirect
)

require github.com/asticode/go-astiav v0.7.1

replace github.com/commaai/cereal v0.0.0 => ./cereal/gen/go

replace capnproto.org/go/capnp/v3 v3.0.0 => ./go-capnproto2

replace github.com/asticode/go-astiav v0.7.1 => ./go-astiav
