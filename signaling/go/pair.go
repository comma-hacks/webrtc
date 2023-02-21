package secureput

import (
	"encoding/json"
	"fmt"

	"github.com/gookit/config/v2"
)

type Pairing struct {
	SecretKey string `json:"secret"`
	UUID      string `json:"uuid"`
}

func (app *SecurePut) GenerateNewPairingQR(width uint8) {
	pairing := Pairing{}
	pairing.SecretKey = GenerateSecretKey()
	pairing.UUID = app.Config.DeviceUUID

	app.Config.DeviceSecret = pairing.SecretKey
	config.Set("secret", pairing.SecretKey)
	DumpConfig()

	qrContent, err := json.Marshal(pairing)
	if err != nil {
		fmt.Println(err)
		return
	}

	err = generate_qr(app.Fs, app.QrCodePath(), string(qrContent), width)
	if err != nil {
		fmt.Println(err)
		return
	}
}
