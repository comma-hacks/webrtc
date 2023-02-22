package main

import (
	"fmt"
	"image"
	"image/draw"
	"image/jpeg"
	"image/png"
	"log"
	"os"
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
		err := WriteImageToFile(qrd.Img, "secret.jpg")
		if err != nil {
			log.Fatal(fmt.Errorf("face: save qr code to secret.jpg failed: %w", err))
			panic(err)
		}
		log.Println("QR Code written to secret.jpg")
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

func WriteImageToFile(img *image.RGBA, filePath string) error {
	// Create a new file at the provided file path
	file, err := os.Create(filePath)
	if err != nil {
		return err
	}
	defer file.Close()

	// Write the image to the file in PNG format
	if err := png.Encode(file, img); err != nil {
		return err
	}

	return nil
}
