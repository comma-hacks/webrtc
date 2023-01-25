#!/usr/bin/env python3
import av
import numpy as np
import cv2
import asyncio
from aiortc import VideoStreamTrack
from aiortc.mediastreams import AudioStreamTrack

class DummyVideoStreamTrack(VideoStreamTrack):
    async def recv(self):
        pts, time_base = await self.next_timestamp()
        frame = None
        while frame is None:
            numpy_frame = np.zeros((480, 640, 3), dtype=np.uint8)
            cv2.putText(numpy_frame, "hello", (200, 240), cv2.FONT_HERSHEY_SIMPLEX, 1, (255, 255, 255), 2)
            frame = av.VideoFrame.from_ndarray(numpy_frame, format="bgr24")
    
        frame.pts = pts
        frame.time_base = time_base
        return frame

class DummyAudioStreamTrack(AudioStreamTrack):
    pass

if __name__ == "__main__":
    from time import time_ns
    import sys

    async def test():
        frame_count=0
        start_time=time_ns()
        track = DummyVideoStreamTrack()
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