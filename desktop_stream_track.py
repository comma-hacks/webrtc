#!/usr/bin/env python3
import av
import asyncio
from aiortc import VideoStreamTrack
import Xlib
import Xlib.display
import os
from desktop_control_interface import DesktopControlInterface

# https://ffmpeg.org/ffmpeg-devices.html#x11grab
class DesktopStreamTrack(VideoStreamTrack):
    def __init__(self):
        super().__init__()
        self.resolution = Xlib.display.Display(os.environ["DISPLAY"]).screen().root.get_geometry()
        self.control_interface = DesktopControlInterface(self.resolution)
        options =  {
            'draw_mouse': '1',
            'i':':0.0+0,0',
            'framerate':'20',
            'video_size': str(self.resolution.width) + "x" + str(self.resolution.height)
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
    
    def handle_action(self, action, data):
        self.control_interface.handle_action(action, data)
            
    def stop(self) -> None:
        super().stop()
        self.control_interface.stop()

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