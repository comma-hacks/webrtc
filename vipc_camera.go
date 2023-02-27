package main

import "github.com/asticode/go-astiav"

type VisionIpcCamera struct {
	prefix string
}

var (
	ROAD      = VisionIpcCamera{prefix: "road"}
	WIDE_ROAD = VisionIpcCamera{prefix: "wideRoad"}
	DRIVER    = VisionIpcCamera{prefix: "driver"}
)

var VisionIpcCameras = []VisionIpcCamera{ROAD, WIDE_ROAD, DRIVER}

func VisionIpcCameraTimeBase() (t astiav.Rational) {
	return astiav.NewRational(1, VisionIpcCameraFrameRate())
}

func VisionIpcCameraFrameRate() int {
	return 20
}

func VisionIpcCameraHeight() int {
	return 1208
}

func VisionIpcCameraWidth() int {
	return 1928
}

func VisionIpcCameraAspectRatio() (t astiav.Rational) {
	return astiav.NewRational(151, 241)
}

func VisionIpcCameraPixelFormat() (f astiav.PixelFormat) {
	return astiav.PixelFormatYuv420P
}
