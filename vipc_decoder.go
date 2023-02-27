package main

import (
	"errors"
	"fmt"
	"log"

	"github.com/asticode/go-astiav"
)

func NewVipcDecoder() (cc *astiav.CodecContext, err error) {
	decCodec := astiav.FindDecoder(astiav.CodecIDHevc)
	if decCodec == nil {
		return nil, errors.New("vision_ipc_track: codec is nil")
	}

	// Alloc codec context
	if cc = astiav.AllocCodecContext(decCodec); cc == nil {
		return nil, errors.New("vision_ipc_track: codec context is nil")
	}

	cc.SetHeight(VisionIpcCameraHeight())
	cc.SetPixelFormat(VisionIpcCameraPixelFormat())
	cc.SetSampleAspectRatio(VisionIpcCameraAspectRatio())
	cc.SetTimeBase(VisionIpcCameraTimeBase())
	cc.SetWidth(VisionIpcCameraWidth())

	// Open codec context
	if err := cc.Open(decCodec, nil); err != nil {
		log.Fatal(fmt.Errorf("vision_ipc_track: opening codec context failed: %w", err))
		return nil, err
	}

	return cc, nil
}
