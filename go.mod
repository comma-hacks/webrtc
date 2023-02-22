module github.com/kfatehi/webrtc-body

go 1.19

require github.com/pebbe/zmq4 v1.2.9

require (
	github.com/google/uuid v1.3.0 // indirect
	github.com/pion/datachannel v1.5.5 // indirect
	github.com/pion/dtls/v2 v2.2.4 // indirect
	github.com/pion/ice/v2 v2.3.0 // indirect
	github.com/pion/interceptor v0.1.12 // indirect
	github.com/pion/logging v0.2.2 // indirect
	github.com/pion/mdns v0.0.7 // indirect
	github.com/pion/randutil v0.1.0 // indirect
	github.com/pion/rtcp v1.2.10 // indirect
	github.com/pion/rtp v1.7.13 // indirect
	github.com/pion/sctp v1.8.6 // indirect
	github.com/pion/sdp/v3 v3.0.6 // indirect
	github.com/pion/srtp/v2 v2.0.12 // indirect
	github.com/pion/stun v0.4.0 // indirect
	github.com/pion/transport/v2 v2.0.1 // indirect
	github.com/pion/turn/v2 v2.1.0 // indirect
	github.com/pion/udp v0.1.4 // indirect
	github.com/pion/webrtc/v3 v3.1.55 // indirect
	github.com/stretchr/testify v1.8.1 // indirect
	golang.org/x/crypto v0.6.0 // indirect
	golang.org/x/net v0.6.0 // indirect
	golang.org/x/sys v0.5.0 // indirect
)

require (
	capnproto.org/go/capnp/v3 v3.0.0
	github.com/asticode/go-astiav v0.7.1
	github.com/commaai/cereal v0.0.0
	golang.org/x/sync v0.0.0-20220722155255-886fb9371eb4 // indirect
)

replace github.com/commaai/cereal v0.0.0 => ./cereal/gen/go

replace capnproto.org/go/capnp/v3 v3.0.0 => ./go-capnproto2

replace github.com/asticode/go-astiav v0.7.1 => ./go-astiav

replace github.com/secureput/signaling v1.0.0 => ./signaling/go
