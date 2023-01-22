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
from aiortc.contrib.signaling import BYE
from secureput.secureput_signaling import SecureputSignaling

from aiortc.contrib.media import MediaBlackhole

# optional, for better performance
# try:
#     import uvloop
# except ImportError:
#     uvloop = None





import tracemalloc

tracemalloc.start(10)


async def heap_snapshot():
    while True:
        snapshot = tracemalloc.take_snapshot()
        top_stats = snapshot.statistics('lineno')

        print("[ Top 10 ]")
        for stat in top_stats[:10]:
            print(stat)
        await asyncio.sleep(10)


cams = ["roadEncodeData","wideRoadEncodeData","driverEncodeData"]
cam = 2

desktop_track = None
recorder = None

def get_desktop_track():
    global desktop_track
    if desktop_track != None:
        desktop_track.stop()
    desktop_track = DesktopStreamTrack()
    return desktop_track

async def signal(pc, signaling):
    global recorder
    
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

            if pc != None and pc.iceConnectionState != "failed":
                print("Closing previous connection")
                await pc.close()
                pc = RTCPeerConnection(configuration=RTCConfiguration([RTCIceServer("stun:stun.secureput.com:3478")]))

            @pc.on("connectionstatechange")
            async def on_connectionstatechange():
                print("Connection state is %s" % pc.iceConnectionState)
                if pc.iceConnectionState == "failed":
                    await pc.close()

            @pc.on('datachannel')
            def on_datachannel(channel):
                print("data channel!")
                @channel.on('message')
                async def on_message(message):
                    data = json.loads(message)
                    if desktop_track:
                        desktop_track.handle_message(data)

            @pc.on("track")
            def on_track(track):
                print("Receiving %s" % track.kind)
                recorder.addTrack(track)


            await pc.setRemoteDescription(obj)
            await recorder.start()
            
            if obj.type == 'offer':
                # TODO: stream the microphone
                audio = None

                # video = VisionIpcTrack(cams[int(cam)], "tici")
                
                # pc.addTrack(video)
                pc.addTrack(get_desktop_track())
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

    recorder = MediaBlackhole()

    if args.verbose:
        logging.basicConfig(level=logging.DEBUG)

    # if uvloop is not None:
    #     asyncio.set_event_loop_policy(uvloop.EventLoopPolicy())

    pc = RTCPeerConnection(configuration=RTCConfiguration([RTCIceServer(args.stun_server)]))
    signaling = SecureputSignaling(args.signaling_server)
    
    coro = signal(pc, signaling)
    # coro = asyncio.gather(watchdog.check_memory(), signal(pc, signaling))
    # coro = asyncio.gather(heap_snapshot(), signal(pc, signaling))


    # run event loop
    loop = asyncio.get_event_loop()
    try:
        loop.run_until_complete(coro)
    except KeyboardInterrupt:
        pass
    finally:
        loop.run_until_complete(recorder.stop())
        loop.run_until_complete(pc.close())
        loop.run_until_complete(signaling.close())
