For the signaling server I am using that of a commercial project I created called SecurePut which is for pasting passwords and other stored text payloads onto Mac, Windows, and Linux computers.

If you're interested in that, you can learn more about the commercial project at the website https://secureput.com/

It is not open source, but in order to facilitate this effort I have open-sourced a Python and Go implementation of the WebRTC signaling component.

The main feature being taken advantage of by using this signaling server implementation is the "free" security and QR-code based "pairing" feature, as well as the ability to reuse the client software I've already built.

When you start the webrtc server it will show you a QR code that you can scan with the SecurePut app. At this point you will see your device and be able to connect.