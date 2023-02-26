package main

import (
	"errors"
	"fmt"
	"log"

	"github.com/asticode/go-astiav"
)

type HEVCDecoder struct {
	decCodecID      astiav.CodecID
	decCodec        *astiav.Codec
	decCodecContext *astiav.CodecContext
}

func NewHEVCDecoder() (decoder *HEVCDecoder, err error) {
	decoder = &HEVCDecoder{}

	//=============
	// HEVC DECODER
	//=============

	decoder.decCodecID = astiav.CodecIDHevc

	// Find decoder
	if decoder.decCodec = astiav.FindDecoder(decoder.decCodecID); decoder.decCodec == nil {
		return nil, errors.New("vision_ipc_track: codec is nil")
	}

	// Alloc codec context
	if decoder.decCodecContext = astiav.AllocCodecContext(decoder.decCodec); decoder.decCodecContext == nil {
		return nil, errors.New("vision_ipc_track: codec context is nil")
	}

	decoder.decCodecContext.SetHeight(decoder.Height())
	decoder.decCodecContext.SetPixelFormat(decoder.PixelFormat())
	decoder.decCodecContext.SetSampleAspectRatio(decoder.AspectRatio())
	decoder.decCodecContext.SetTimeBase(decoder.TimeBase())
	decoder.decCodecContext.SetWidth(decoder.Width())

	// Open codec context
	if err := decoder.decCodecContext.Open(decoder.decCodec, nil); err != nil {
		log.Fatal(fmt.Errorf("vision_ipc_track: opening codec context failed: %w", err))
		return nil, err
	}

	return decoder, nil
}

func (decoder *HEVCDecoder) Close() {
	decoder.decCodecContext.Free()
}

func (decoder *HEVCDecoder) TimeBase() (t astiav.Rational) {
	return astiav.NewRational(1, decoder.FrameRate())
}

func (decoder *HEVCDecoder) FrameRate() int {
	return 20
}

func (decoder *HEVCDecoder) Height() int {
	return 1208
}

func (decoder *HEVCDecoder) Width() int {
	return 1928
}

func (decoder *HEVCDecoder) AspectRatio() (t astiav.Rational) {
	return astiav.NewRational(151, 241)
}

func (decoder *HEVCDecoder) PixelFormat() (f astiav.PixelFormat) {
	return astiav.PixelFormatYuv420P
}
