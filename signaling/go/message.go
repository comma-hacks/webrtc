package secureput

import (
	"encoding/json"

	"github.com/pion/webrtc/v3"
)

type Message struct {
	PayloadType    string          `json:"type"`
	PayloadContent json.RawMessage `json:"payload"`
}

type SDPMessage struct {
	PayloadType    string                    `json:"type"`
	PayloadContent webrtc.SessionDescription `json:"payload"`
}

type ICECandidateInit struct {
	SDP           string  `json:"sdp"`
	SDPMid        *string `json:"sdpMid"`
	SDPMLineIndex *uint16 `json:"sdpMLineIndex"`
}

type ICEMessage struct {
	PayloadType    string           `json:"type"`
	PayloadContent ICECandidateInit `json:"payload"`
}

const SessionDescriptionType = "SessionDescription"
const IceCandidateType = "IceCandidate"
const IdentificationType = "identify-target"

type IdentificationPayload struct {
	Name        string `json:"name"`
	DeviceUUID  string `json:"device"`
	AccountUUID string `json:"account"`
}

type IdentificationMessage struct {
	PayloadType    string                `json:"type"`
	PayloadContent IdentificationPayload `json:"payload"`
}

const WrappedType = "wrapped"

type WrappedPayload struct {
	From string `json:"from"`
	Data string `json:"data"`
}

type WrappedMessage struct {
	PayloadType    string         `json:"type"`
	PayloadContent WrappedPayload `json:"payload"`
}

const ForwardWrappedType = "forward-wrapped"

type ForwardWrappedMessage struct {
	Type string `json:"type"`
	To   string `json:"to"`
	Body string `json:"body"`
}

const ClaimType = "claim"

type ClaimPayload struct {
	AccountUUID string `json:"account"`
}

type ClaimMessage struct {
	PayloadType    string       `json:"type"`
	PayloadContent ClaimPayload `json:"payload"`
}
