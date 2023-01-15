#!/usr/bin/env python3
import av
import asyncio
from aiortc import VideoStreamTrack
import numpy

# https://ffmpeg.org/ffmpeg-devices.html#x11grab
class DesktopStreamTrack(VideoStreamTrack):
    def __init__(self):
        super().__init__()
        options =  {
            'i':':0.0+0,0',
            'framerate':'20',
            'video_size': '1920x1080'
        }
        self.container = av.open(':0', format='x11grab', options=options)

    async def recv(self):
        pts, time_base = await self.next_timestamp()
        frame = None
        while frame is None:
            frame = next(self.container.decode(video=0))
    
        frame.pts = pts
        frame.time_base = time_base
        return frame

if __name__ == "__main__":
    from time import time_ns
    import sys

    async def test():
        frame_count=0
        start_time=time_ns()
        track = DesktopStreamTrack()
        while True:
            await track.recv()
            now = time_ns()
            playtime = now - start_time
            playtime_sec = playtime * 0.000000001
            if playtime_sec >= 1:
                print(f'fps: {frame_count}')
                frame_count = 0
                start_time = time_ns()
            else:
                frame_count+=1

    # Run event loop
    loop = asyncio.new_event_loop()
    try:
        loop.run_until_complete(test())
    except KeyboardInterrupt:
        sys.exit(0)