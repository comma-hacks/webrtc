#!/usr/bin/env python3
import av
import asyncio
from aiortc import VideoStreamTrack
import Xlib
import Xlib.display
import os
import pyautogui
import numpy
import evdev
import keymap

pyautogui.FAILSAFE = False

# https://ffmpeg.org/ffmpeg-devices.html#x11grab
class DesktopStreamTrack(VideoStreamTrack):
    def __init__(self):
        super().__init__()
        self.resolution = Xlib.display.Display(os.environ["DISPLAY"]).screen().root.get_geometry()
        options =  {
            'draw_mouse': '1',
            'i':':0.0+0,0',
            'framerate':'20',
            'video_size': str(self.resolution.width) + "x" + str(self.resolution.height)
        }
        self.container = av.open(':0', format='x11grab', options=options)
        self.ui = evdev.UInput()
        self.valid_actions = ["keyboard", "click", "rightclick", "mousemove", "joystick", "paste"]

    async def recv(self):
        pts, time_base = await self.next_timestamp()
        frame = None
        while frame is None:
            frame = next(self.container.decode(video=0))
    
        frame.pts = pts
        frame.time_base = time_base
        return frame

    def handle_action(self, action, data):
        if action == "mousemove":
            x = numpy.interp(data["cursorPositionX"], (0, data["displayWidth"]), (0, self.resolution.width))
            y = numpy.interp(data["cursorPositionY"], (0, data["displayHeight"]), (0, self.resolution.height))
            pyautogui.moveTo(x, y, _pause=False)
        # elif action == "joystick":
        #     x = numpy.interp(data["x"], (-38, 38), (0, self.resolution.width))
        #     y = numpy.interp(data["y"], (-38, 38), (self.resolution.height, 0))
        #     print(f'{data["y"]} {self.resolution.height} {y}')
        #     pyautogui.moveTo(x, y, _pause=False)
        elif action == "click":
            pyautogui.click()
        elif action == "rightclick":
            pyautogui.rightClick()
        elif action == "keyboard":
            try:
                # keymap.reload()
                osKey = keymap.iOStoLinux[data["key"]]
                self.ui.write(evdev.ecodes.EV_KEY, osKey, data["direction"])
                self.ui.syn()
            except KeyError:
                print(f"Unknown key: {data['key']}")
        elif action == "paste":
            # might as well support the secureput protocol completely
            pyautogui.write(data["payload"]["string"])
            
    def stop(self) -> None:
        super().stop()
        self.ui.close()

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