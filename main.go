package main

import (
	"fmt"
	"log"
	"strings"

	"github.com/asticode/go-astiav"
)

func main() {
	track, err := NewVisionIpcTrack("roadEncodeData")
	if err != nil {
		log.Fatal(fmt.Errorf("main: creating track failed: %w", err))
	}
	defer track.Stop()

	// Handle ffmpeg logs
	astiav.SetLogLevel(astiav.LogLevelError)
	astiav.SetLogCallback(func(l astiav.LogLevel, fmt, msg, parent string) {
		log.Printf("ffmpeg log: %s (level: %d)\n", strings.TrimSpace(msg), l)
	})

	go track.Start()

	count := 0
	for frame := range track.frameChannel {
		// Do something with decoded frame
		// log.Printf("%d, new frame: width: %d", track.networkLatency, frame.Width())
		frame.Pts()
		count++

		if count > 10 {
			track.Stop()
		}
	}
}
