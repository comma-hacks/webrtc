package secureput

// https://gist.github.com/locked/b066aa1ddeb2b:e855e

import (
	"bytes"
	"encoding/json"
	"log"
	"strings"

	"github.com/gorilla/websocket"
	"github.com/pion/webrtc/v3"
)

const (
	Identified = iota
	Connected
	CloseError
	UnexpectedCloseError
	ConnectionTimeout
)

var peerConnection *webrtc.PeerConnection

func (ap *SecurePut) SignalMessageHandler(message *Message, signalClient *websocket.Conn) {

	switch message.PayloadType {
	case SessionDescriptionType:
		var err error

		if peerConnection != nil {
			peerConnection.Close()
			peerConnection = nil
		}

		// Prepare the configuration
		var webrtcConfig = webrtc.Configuration{
			ICEServers: []webrtc.ICEServer{
				{
					URLs: StunServers,
				},
			},
		}

		if peerConnection, err = webrtc.NewPeerConnection(webrtcConfig); err != nil {
			log.Println("new peer connection -> error:", err)
			return
		}

		var offer webrtc.SessionDescription
		if err := json.Unmarshal(message.PayloadContent, &offer); err != nil {
			log.Fatal(err)
		}
		log.Println("got offer")

		if err = peerConnection.SetRemoteDescription(offer); err != nil {
			panic(err)
		}

		answer, err := peerConnection.CreateAnswer(nil)
		if err != nil {
			panic(err)
		} else if err = peerConnection.SetLocalDescription(answer); err != nil {
			panic(err)
		}

		ap.EncryptingWriteJSON(signalClient, SDPMessage{
			PayloadType:    SessionDescriptionType,
			PayloadContent: *peerConnection.LocalDescription(),
		})

		log.Println("sent a reply")

		// Set the handler for ICE connection state
		// This will notify you when the peer has connected/disconnected
		peerConnection.OnICEConnectionStateChange(func(connectionState webrtc.ICEConnectionState) {
			log.Printf("ICE Connection State has changed: %s\n", connectionState.String())
		})

		peerConnection.OnDataChannel(func(d *webrtc.DataChannel) {
			d.OnOpen(func() {
				log.Println("peerconn data channel open")
			})
			d.OnMessage(func(msg webrtc.DataChannelMessage) {
				var message Message
				err := json.Unmarshal(msg.Data, &message)
				if err != nil {
					log.Println("data channel message unmarshall error:", err)
					return
				}
				switch message.PayloadType {
				case WrappedType:
					var wp WrappedPayload
					if err := json.Unmarshal(message.PayloadContent, &wp); err != nil {
						log.Fatal(err)
					}
					data, err := decrypt([]byte(ap.Config.DeviceSecret), wp.Data)
					if err != nil {
						log.Println("failed to decrypt")
						log.Println(err)
						return
					}
					var msg Message
					decoder := json.NewDecoder(strings.NewReader(string(data)))
					if err = decoder.Decode(&msg); err != nil {
						log.Println("failed to decode decrypted message", string(data))
						log.Println(err)
						return
					}
					switch msg.PayloadType {
					default:
						log.Println("unexpected unwrapped message on data channel")
					}
				default:
					log.Println("unexpected message on data channel")
				}
			})
		})

		peerConnection.OnICECandidate(func(i *webrtc.ICECandidate) {
			if i != nil {
				// check(offerPC.AddICECandidate(i.ToJSON()))
				ic := i.ToJSON()
				ap.EncryptingWriteJSON(signalClient, ICEMessage{
					PayloadType: IceCandidateType,
					PayloadContent: ICECandidateInit{
						SDP:           ic.Candidate,
						SDPMid:        ic.SDPMid,
						SDPMLineIndex: ic.SDPMLineIndex,
					},
				})
				log.Println("sent ice candidate")
			}
		})

	case IceCandidateType:
		var ic ICECandidateInit
		if err := json.Unmarshal(message.PayloadContent, &ic); err != nil {
			log.Fatal(err)
		}
		log.Println("got ice candidate")
		if peerConnection != nil {
			peerConnection.AddICECandidate(webrtc.ICECandidateInit{
				Candidate:     ic.SDP,
				SDPMid:        ic.SDPMid,
				SDPMLineIndex: ic.SDPMLineIndex,
			})
		}
	case WrappedType:
		var wp WrappedPayload
		if err := json.Unmarshal(message.PayloadContent, &wp); err != nil {
			log.Fatal(err)
		}
		data, err := decrypt([]byte(ap.Config.DeviceSecret), wp.Data)
		if err != nil {
			log.Println("failed to decrypt")
			log.Println(err)
			return
		}
		var msg Message
		decoder := json.NewDecoder(strings.NewReader(string(data)))
		if err = decoder.Decode(&msg); err != nil {
			// user revoked access
			log.Println("Received a message that could not be decrypted. Sender does not have the new key.")
			return
		}
		ap.SignalMessageHandler(&msg, signalClient)
	case ClaimType:
		var claim ClaimPayload
		if err := json.Unmarshal(message.PayloadContent, &claim); err != nil {
			log.Fatal(err)
		}
		log.Printf("validated a claim from %s\n", claim.AccountUUID)
		ap.PairChannel <- claim.AccountUUID
		signalClient.WriteJSON(IdentificationMessage{
			PayloadType: IdentificationType,
			PayloadContent: IdentificationPayload{
				Name:        ap.GetName(),
				DeviceUUID:  ap.Config.DeviceUUID,
				AccountUUID: claim.AccountUUID,
			},
		})
	default:
		log.Printf("Whoops. unhandled message type %s\n", message.PayloadType)
	}
}

func (ap *SecurePut) EncryptingWriteJSON(c *websocket.Conn, v interface{}) error {
	msg := ForwardWrappedMessage{}
	msg.To = ap.Config.AccountUUID
	msg.Type = ForwardWrappedType
	w, err := c.NextWriter(websocket.TextMessage)
	if err != nil {
		return err
	}
	// Encode the plaintext to a string
	buf := new(bytes.Buffer)
	err0 := json.NewEncoder(buf).Encode(v)
	if err0 != nil {
		return err0
	}
	ciphertext, err3 := encrypt([]byte(ap.Config.DeviceSecret), buf.Bytes())
	if err3 != nil {
		return err3
	}
	// install the ciphertext into the payload
	msg.Body = ciphertext

	// encode as json and write to socket
	err1 := json.NewEncoder(w).Encode(msg)
	err2 := w.Close()
	if err1 != nil {
		return err1
	}
	return err2
}

func (ap *SecurePut) Signal() {
	u := SignalServer
	var signalClient *websocket.Conn
	for signalClient == nil {
		log.Printf("connecting to %s", u.String())
		c, _, err := websocket.DefaultDialer.Dial(u.String(), nil)
		if err != nil {
			ap.SignalStatusChannel <- ConnectionTimeout
			return
		}
		signalClient = c
		ap.SignalStatusChannel <- Connected

		signalClient.WriteJSON(IdentificationMessage{
			PayloadType: IdentificationType,
			PayloadContent: IdentificationPayload{
				Name:        ap.GetName(),
				DeviceUUID:  ap.Config.DeviceUUID,
				AccountUUID: ap.Config.AccountUUID,
			},
		})
		ap.SignalStatusChannel <- Identified
	}

	defer signalClient.Close()

	for {
		var message Message
		if err := signalClient.ReadJSON(&message); err != nil {
			log.Println("recv error:", err)
			if websocket.IsCloseError(err) {
				ap.SignalStatusChannel <- CloseError
				return
			}
			if websocket.IsUnexpectedCloseError(err) {
				ap.SignalStatusChannel <- UnexpectedCloseError
				return
			}
		}
		ap.SignalMessageHandler(&message, signalClient)
	}
}
