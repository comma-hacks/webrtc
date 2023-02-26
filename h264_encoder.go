package main

import (
	"errors"
	"fmt"

	"github.com/asticode/go-astiav"
)

type H264Encoder struct {
	encCodecID      astiav.CodecID
	encCodec        *astiav.Codec
	encCodecContext *astiav.CodecContext
}

func NewH264Encoder() (encoder *H264Encoder, err error) {
	encoder = &H264Encoder{}

	//=============
	// H264 ENCODER
	//=============

	// Get codec id
	encoder.encCodecID = astiav.CodecIDH264

	// Find encoder
	if encoder.encCodec = astiav.FindEncoder(encoder.encCodecID); encoder.encCodec == nil {
		err = errors.New("vision_ipc_track: codec is nil")
		return
	}

	// Alloc codec context
	if encoder.encCodecContext = astiav.AllocCodecContext(encoder.encCodec); encoder.encCodecContext == nil {
		err = errors.New("main: codec context is nil")
		return
	}

	encoder.encCodecContext.SetHeight(encoder.Height())
	encoder.encCodecContext.SetFramerate(astiav.NewRational(20, 1))
	encoder.encCodecContext.SetPixelFormat(encoder.PixelFormat())
	encoder.encCodecContext.SetBitRate(500_000)
	// encoder.encCodecContext.SetGopSize(0)
	encoder.encCodecContext.SetSampleAspectRatio(encoder.AspectRatio())
	encoder.encCodecContext.SetTimeBase(encoder.TimeBase())
	encoder.encCodecContext.SetWidth(encoder.Width())
	encoder.encCodecContext.SetStrictStdCompliance(astiav.StrictStdComplianceExperimental)

	// Update flags
	flags := encoder.encCodecContext.Flags()
	flags = flags.Add(astiav.CodecContextFlagLowDelay)
	encoder.encCodecContext.SetFlags(flags)

	// TOOD possibly PR to go-astiav with this SetOpt function
	// av_opt_set(mCodecContext->priv_data, "preset", "ultrafast", 0);
	// av_opt_set(mCodecContext->priv_data, "tune", "zerolatency", 0);
	encoder.encCodecContext.SetOpt("preset", "ultrafast", 0)
	encoder.encCodecContext.SetOpt("tune", "zerolatency", 0)
	// from https://trac.ffmpeg.org/ticket/3354
	// interesting: https://github.com/giorgisio/goav/blob/master/avcodec/context.go#L190

	// Open codec context
	if err = encoder.encCodecContext.Open(encoder.encCodec, nil); err != nil {
		err = fmt.Errorf("encoder: opening codec context failed: %w", err)
		return
	}

	return encoder, nil
}

func (encoder *H264Encoder) Close() {
	encoder.encCodecContext.Free()
}

func (encoder *H264Encoder) TimeBase() (t astiav.Rational) {
	return astiav.NewRational(1, encoder.FrameRate())
}

func (encoder *H264Encoder) FrameRate() int {
	return 20
}

func (encoder *H264Encoder) Height() int {
	return 1208
}

func (encoder *H264Encoder) Width() int {
	return 1928
}

func (encoder *H264Encoder) AspectRatio() (t astiav.Rational) {
	return astiav.NewRational(151, 241)
}

func (encoder *H264Encoder) PixelFormat() (f astiav.PixelFormat) {
	return astiav.PixelFormatYuv420P
}
