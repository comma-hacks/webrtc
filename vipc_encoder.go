package main

import (
	"errors"
	"fmt"

	"github.com/asticode/go-astiav"
)

func NewVipcEncoder(dcc *astiav.CodecContext) (cc *astiav.CodecContext, err error) {
	encCodec := astiav.FindEncoder(astiav.CodecIDH264)
	if encCodec == nil {
		return nil, errors.New("vision_ipc_track: codec is nil")
	}

	if cc = astiav.AllocCodecContext(encCodec); cc == nil {
		return nil, errors.New("main: codec context is nil")
	}

	cc.SetHeight(dcc.Height())
	cc.SetFramerate(dcc.Framerate())
	cc.SetPixelFormat(dcc.PixelFormat())
	cc.SetBitRate(200_000)
	// cc.SetGopSize(0)
	cc.SetSampleAspectRatio(dcc.SampleAspectRatio())
	cc.SetTimeBase(dcc.TimeBase())
	cc.SetWidth(dcc.Width())
	cc.SetStrictStdCompliance(astiav.StrictStdComplianceExperimental)

	// Update flags
	flags := cc.Flags()
	flags = flags.Add(astiav.CodecContextFlagLowDelay)
	cc.SetFlags(flags)

	// TOOD possibly PR to go-astiav with this SetOpt function
	// av_opt_set(mCodecContext->priv_data, "preset", "ultrafast", 0);
	// av_opt_set(mCodecContext->priv_data, "tune", "zerolatency", 0);
	cc.SetOpt("preset", "ultrafast", 0)
	cc.SetOpt("tune", "zerolatency", 0)
	// from https://trac.ffmpeg.org/ticket/3354
	// interesting: https://github.com/giorgisio/goav/blob/master/avcodec/context.go#L190

	// Open codec context
	err = cc.Open(encCodec, nil)
	if err != nil {
		return nil, fmt.Errorf("encoder: opening codec context failed: %w", err)
	}

	return cc, nil
}
