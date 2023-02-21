package secureput

import (
	"image"
	"image/draw"
	"image/jpeg"
	"log"
	"net/url"
	"os"
	"time"

	"github.com/google/uuid"
	"github.com/gookit/config/v2"
	"github.com/spf13/afero"
)

var SignalServer = url.URL{Scheme: "wss", Host: "signal.secureput.com", Path: "/"}
var StunServers = []string{"stun:stun.secureput.com:3478"}

type IGui interface {
	Changed()
	Close()
	Show()
}

type AppConfig struct {
	DeviceSecret string
	DeviceUUID   string
	AccountUUID  string
}

const (
	PipeMode int = iota
	DaemonMode
)

type SecurePut struct {
	OutputMode          int
	Fs                  afero.Fs
	Config              AppConfig
	QRImage             *image.RGBA
	PairChannel         chan string
	SignalStatusChannel chan int
	Gui                 IGui
}

func Create() SecurePut {
	ap := SecurePut{}
	ap.Fs = afero.NewMemMapFs()
	ap.PairChannel = make(chan string)
	ap.SignalStatusChannel = make(chan int)
	InitConfig()
	ap.Config.AccountUUID = config.String("account")
	ap.Config.DeviceSecret = config.String("secret")
	ap.Config.DeviceUUID = config.String("uuid")
	if ap.Config.DeviceUUID == "" {
		if err := config.Set("uuid", uuid.New().String()); err != nil {
			panic(err)
		}
		ap.Config.DeviceUUID = config.String("uuid")
	}
	return ap
}

func (ap *SecurePut) Paired() bool {
	return ap.Config.AccountUUID != ""
}

func (ap *SecurePut) QrCodePath() string {
	return "qr.jpg"
}

func (ap *SecurePut) GenPairInfo() {
	ap.GenerateNewPairingQR(6)
	fh, err := ap.Fs.Open(ap.QrCodePath())
	if err == nil {
		defer fh.Close()
		img, _ := jpeg.Decode(fh)
		ap.QRImage = image.NewRGBA(img.Bounds())
		draw.Draw(ap.QRImage, img.Bounds(), img, image.Point{}, draw.Src)
	}
}

func (ap *SecurePut) ClearPairing() {
	config.Set("account", "")
	DumpConfig()
	ap.Config.AccountUUID = config.String("account")
	ap.Gui.Changed()
}

func (ap *SecurePut) GetName() string {
	hostname, err := os.Hostname()
	if err != nil {
		panic(err)
	}
	return hostname
}

func (ap *SecurePut) RunDaemonMode() {
	ap.OutputMode = DaemonMode
	log.Println("Daemon mode")
	var account string
	var status int
	go ap.Signal()
	for {
		log.Println("Select")

		select {
		case account = <-ap.PairChannel:
			log.Println("Pair channel message", account)
			ap.Config.AccountUUID = account
			ap.Gui.Changed()
			config.Set("account", account)
			DumpConfig()
		case status = <-ap.SignalStatusChannel:
			log.Println("Status info")
			switch status {
			case Identified:
				log.Println("Identified")
			case Connected:
				log.Println("Connected")
			case ConnectionTimeout:
				log.Println("Timeout")
				time.Sleep(1_000_000_000)
				go ap.Signal()
			case CloseError, UnexpectedCloseError:
				log.Println("Signaling Error")
				go ap.Signal()
			}
		}
	}
}
