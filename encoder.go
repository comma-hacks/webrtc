package main

import (
	"errors"
	"fmt"
	"log"

	"github.com/asticode/go-astiav"
	"github.com/asticode/go-astikit"
)

type EncoderStream struct {
	encCodec        *astiav.Codec
	encCodecContext *astiav.CodecContext
	encPkt          *astiav.Packet
}

type Encoder struct {
	Stream *EncoderStream
	Closer *astikit.Closer
}

type EncoderParams struct {
	Height      int
	Width       int
	TimeBase    astiav.Rational
	AspectRatio astiav.Rational
	PixelFormat astiav.PixelFormat
}

func NewEncoder(params EncoderParams) (ts *Encoder, err error) {
	ts = &Encoder{}
	s := &EncoderStream{}
	c := astikit.NewCloser()
	ts.Closer = c
	ts.Stream = s

	// Get codec id
	codecID := astiav.CodecIDMpeg4

	// Find encoder
	if s.encCodec = astiav.FindEncoder(codecID); s.encCodec == nil {
		err = errors.New("main: codec is nil")
		return
	}

	// Alloc codec context
	if s.encCodecContext = astiav.AllocCodecContext(s.encCodec); s.encCodecContext == nil {
		err = errors.New("main: codec context is nil")
		return
	}
	c.Add(s.encCodecContext.Free)

	s.encCodecContext.SetHeight(params.Height)
	s.encCodecContext.SetPixelFormat(params.PixelFormat)
	s.encCodecContext.SetBitRate(1000)
	s.encCodecContext.SetSampleAspectRatio(params.AspectRatio)
	s.encCodecContext.SetTimeBase(params.TimeBase)
	s.encCodecContext.SetWidth(params.Width)

	// Open codec context
	if err = s.encCodecContext.Open(s.encCodec, nil); err != nil {
		err = fmt.Errorf("encoder: opening codec context failed: %w", err)
		return
	}

	// Allocate packet
	s.encPkt = astiav.AllocPacket()
	c.Add(s.encPkt.Free)

	return ts, nil
}

func (ts *Encoder) Close() {
	ts.Closer.Close()
}

func (ts *Encoder) Encode(f *astiav.Frame) (pkt *astiav.Packet, err error) {
	s := ts.Stream

	// Unref packet
	s.encPkt.Unref()

	// Send frame
	if err = s.encCodecContext.SendFrame(f); err != nil {
		err = fmt.Errorf("encoder: sending frame failed: %w", err)
		return nil, err
	}

	if err := s.encCodecContext.ReceivePacket(s.encPkt); err != nil {
		log.Fatal(fmt.Errorf("encoder: receiving frame failed: %w", err))
		return nil, err
	}

	return s.encPkt, nil
}
