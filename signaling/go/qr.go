package secureput

import (
	"log"

	qrcode_memwriter "github.com/kfatehi/go-qrcode-memwriter"
	"github.com/spf13/afero"
	"github.com/yeqown/go-qrcode/v2"
	"github.com/yeqown/go-qrcode/writer/standard"
)

func generate_qr(AppFs afero.Fs, path string, content string, width uint8) error {
	qrc, err := qrcode.New(content)
	if err != nil {
		log.Printf("could not generate QRCode: %v", err)
		return err
	}

	w, err := qrcode_memwriter.New(AppFs, path, standard.WithQRWidth(width))
	if err != nil {
		log.Printf("standard.New failed: %v", err)
		return err
	}

	// save file
	if err = qrc.Save(w); err != nil {
		log.Printf("could not save image: %v", err)
		return err
	}

	return nil
}
