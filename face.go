package main

import (
	"image"
	"image/draw"
	"image/jpeg"
	"log"
	"secureput"
)

type Face struct {
	app *secureput.SecurePut
}

type QrDisplay struct {
	Img          *image.RGBA
	gui          *Face
	Icon         *image.RGBA
	GeneratingQR bool
}

func (qrd *QrDisplay) GenPairInfo() {
	qrd.GeneratingQR = true
	qrd.gui.app.GenerateNewPairingQR(6)
	fh, err := qrd.gui.app.Fs.Open(qrd.gui.app.QrCodePath())
	if err == nil {
		defer fh.Close()
		img, _ := jpeg.Decode(fh)
		qrd.Img = image.NewRGBA(img.Bounds())
		draw.Draw(qrd.Img, img.Bounds(), img, image.Point{}, draw.Src)
		qrd.GeneratingQR = false
		log.Println("QR Code is in memory right now")
	}
}

func (gui *Face) Show() {
	log.Println("Showing GUI")
	qrd := QrDisplay{gui: gui}
	qrd.GenPairInfo()
}

func (gui *Face) Changed() {
	log.Println("GUI changed")
}

func (gui *Face) Close() {
	log.Println("GUI closed")
}
