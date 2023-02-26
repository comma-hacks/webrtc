package main

import (
	"github.com/asticode/go-astiav"
)

type VIPCTranscoder struct {
	decoder *HEVCDecoder
	encoder *H264Encoder
}

type VIPCTranscoderStream struct {
	decPkt   *astiav.Packet
	decFrame *astiav.Frame
	encPkt   *astiav.Packet
}

func NewVIPCTranscoder(decoder *HEVCDecoder, encoder *H264Encoder) (transcoder *VIPCTranscoder, err error) {
	transcoder = &VIPCTranscoder{
		decoder: decoder,
		encoder: encoder,
	}

	return transcoder, nil
}
