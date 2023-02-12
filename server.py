#!/usr/bin/env python
import argparse
import asyncio
import json
import logging
import os
import ssl
from typing import OrderedDict
from aiortc import RTCPeerConnection, RTCSessionDescription, RTCIceCandidate, RTCRtpCodecCapability, RTCConfiguration, RTCIceServer
from compressed_vipc_track import VisionIpcTrack
from desktop_stream_track import DesktopStreamTrack
from dummy_streams import DummyAudioStreamTrack, DummyVideoStreamTrack

from aiortc.contrib.signaling import BYE
from secureput.secureput_signaling import SecureputSignaling

from aiortc.contrib.media import MediaBlackhole
import requests
import numpy
# optional, for better performance
try:
    import uvloop
except ImportError:
    uvloop = None


def webjoystick(x, y):
    outx = (1, -1)
    outy = (-0.5, 0.5)
    if current_track_name == "cam3":
        # outx = (-1, 1)
        outy = (0.5, -0.5)
    x = numpy.interp(x, (-150, 150), outx)
    y = numpy.interp(y, (-150, 150), outy)
    request_url = f"http://tici:5000/control/{x:.4f}/{y:.4f}"
    print(request_url)
    requests.get(request_url)


pc = None
cams = ["roadEncodeData","wideRoadEncodeData","driverEncodeData"]
cam = 2

video_sender = None
desktop_track = None
recorder = None

def get_desktop_track():
    global desktop_track
    if desktop_track != None:
        desktop_track.stop()
    desktop_track = DesktopStreamTrack()
    return desktop_track

current_track_name = None
current_track = None
track_map = {
    "dummy": lambda : DummyVideoStreamTrack(),
    "pc": lambda : get_desktop_track(),
    "cam1": lambda : VisionIpcTrack(cams[int(0)], "tici"),
    "cam2": lambda : VisionIpcTrack(cams[int(1)], "tici"),
    "cam3": lambda : VisionIpcTrack(cams[int(2)], "tici")
}

async def change_tracks(name):
    global current_track_name
    global current_track
    global video_sender
    if current_track_name != name:
        mktrack = track_map[name]
        track = mktrack()
        if current_track_name == None:
            video_sender = pc.addTrack(track)
        else:
            print(f"Changing track to {name}")
            video_sender.replaceTrack(track)
        current_track = track
        current_track_name = name
    return current_track


async def signal(signaling):
    global pc
    global recorder
    global current_track
    global current_track_name
    
    await signaling.connect()
    print("Connected to signaling server")
    while True:
        obj = await signaling.receive()

        # The peer trickles, but aiortc doesn't https://github.com/aiortc/aiortc/issues/227
        # > aioice, the library which aiortc uses for ICE does not trickle ICE candidates:
        # > you get all the candidates in one go. As such once you have called setLocalDescription()
        # > for your offer or answer, all your ICE candidates are listed in pc.localDescription.
        if isinstance(obj, RTCIceCandidate):
            pc.addIceCandidate(obj)

        if isinstance(obj, RTCSessionDescription):
            print("Got SDP")
            
            if pc != None:
                await pc.close()
                current_track = None
                current_track_name = None
                print("Closed previous peer connection")
            
            pc = RTCPeerConnection(configuration=RTCConfiguration([RTCIceServer("stun:stun.secureput.com:3478")]))

            if recorder != None:
                await recorder.stop()
                print("Stopped previous recorder")

            recorder = MediaBlackhole()
                        

            @pc.on("connectionstatechange")
            async def on_connectionstatechange():
                print("Connection state is %s" % pc.iceConnectionState)
                if pc.iceConnectionState == "failed":
                    print("ICE connection failed.")
                    if pc != None:
                        await pc.close()
                        pc = None
                    if recorder != None:
                        await recorder.stop()
                        recorder = None
                   

            @pc.on('datachannel')
            def on_datachannel(channel):
                print("data channel!")
                @channel.on('message')
                async def on_message(message):
                    data = json.loads(message)
                    if "type" in data:
                        if data["type"] == "wrapped":
                            data = json.loads(signaling.decrypt(data["payload"]["data"]))
                        elif data["type"] == "request_track":
                            if data["name"] in track_map:
                                await change_tracks(data["name"])
                            else:
                                print("unknown track requested")
                                print(data)
                        elif data["type"] == "joystick":
                            webjoystick(data["x"], data["y"])
                        elif desktop_track and desktop_track.input.supports(data["type"]):
                            desktop_track.input.handle_action(data["type"], data)
                        else:
                            print("ignored message")
                            print(data)
                    else:
                        print("unsupported message")
                        print(data)

            @pc.on("track")
            def on_track(track):
                print("Receiving %s" % track.kind)
                recorder.addTrack(track)


            await pc.setRemoteDescription(obj)
            await recorder.start()
            
            if obj.type == 'offer':
                # TODO: stream the microphone
                audio = None

                await change_tracks("pc")

                answer = await pc.createAnswer()
                await pc.setLocalDescription(answer)
                await signaling.send(pc.localDescription)

if __name__ == "__main__":
    parser = argparse.ArgumentParser(description="Comma Body WebRTC Service")
    parser.add_argument("--addr", default='tici', help="Address of comma three")

    # Not implemented (yet?). Geo already made the PoC for this, it should be possible.
    # parser.add_argument("--nvidia", action="store_true", help="Use nvidia instead of ffmpeg")

    parser.add_argument("--signaling-server", default="wss://signal.secureput.com", help="Signaling server to use")
    parser.add_argument("--stun-server", default="stun:stun.secureput.com:3478", help="STUN server to use")
    parser.add_argument("--verbose", "-v", action="count")
    args = parser.parse_args()

    if args.verbose:
        logging.basicConfig(level=logging.DEBUG)

    if uvloop is not None:
        asyncio.set_event_loop_policy(uvloop.EventLoopPolicy())

    signaling = SecureputSignaling(args.signaling_server)
    
    coro = signal(signaling)
    # coro = asyncio.gather(watchdog.check_memory(), signal(pc, signaling))
    # coro = asyncio.gather(heap_snapshot(), signal(pc, signaling))


    # run event loop
    loop = asyncio.get_event_loop()
    try:
        loop.run_until_complete(coro)
    except KeyboardInterrupt:
        pass
    finally:
        loop.run_until_complete(signaling.close())
