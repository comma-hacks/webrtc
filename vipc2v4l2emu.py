from compressed_vipc_track import VisionIpcTrack, W, H
import asyncio
import pyvirtualcam
import sys

cam = pyvirtualcam.Camera(width=W, height=H, fps=20)

def work(frame):
    frame = frame.to_ndarray(format='bgr24')
    cam.send(frame)
    cam.sleep_until_next_frame()
        
if __name__ == "__main__":
    from time import time_ns
    import sys

    async def test():
        frame_count=0
        start_time=time_ns()
        track = VisionIpcTrack("roadEncodeData", "tici")
        while True:
            frame = await track.recv()
            work(frame)
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
