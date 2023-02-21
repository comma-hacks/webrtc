#!/usr/bin/env python3
import av
import os
import sys
import argparse
import numpy as np
import multiprocessing
import time

import cereal.messaging as messaging

W, H = 1928, 1208
V4L2_BUF_FLAG_KEYFRAME = 8

def decoder(addr, sock_name, vipc_server, vst, nvidia):
  print("start decoder for %s" % sock_name)
  codec = av.CodecContext.create("hevc", "r")

  os.environ["ZMQ"] = "1"
  messaging.context = messaging.Context()
  sock = messaging.sub_sock(sock_name, None, addr=addr, conflate=False)
  cnt = 0
  last_idx = -1
  seen_iframe = False

  time_q = []
  while 1:
    msgs = messaging.drain_sock(sock, wait_for_one=True)
    for evt_context in msgs:
      with evt_context as evt:
        evta = getattr(evt, evt.which())
        if evta.idx.encodeId != 0 and evta.idx.encodeId != (last_idx+1):
          print("DROP PACKET!")
        last_idx = evta.idx.encodeId
        if not seen_iframe and not (evta.idx.flags & V4L2_BUF_FLAG_KEYFRAME):
          print("waiting for iframe")
          continue
        time_q.append(time.monotonic())
        network_latency = (int(time.time()*1e9) - evta.unixTimestampNanos)/1e6
        frame_latency = ((evta.idx.timestampEof/1e9) - (evta.idx.timestampSof/1e9))*1000
        process_latency = ((evt.logMonoTime/1e9) - (evta.idx.timestampEof/1e9))*1000

        # put in header (first)
        if not seen_iframe:
          codec.decode(av.packet.Packet(evta.header))
          seen_iframe = True

      
        frames = codec.decode(av.packet.Packet(evta.data))
        if len(frames) == 0:
          print("DROP SURFACE")
          continue
        assert len(frames) == 1
        img_yuv = frames[0].to_ndarray(format=av.video.format.VideoFormat('yuv420p')).flatten()
        uv_offset = H*W
        y = img_yuv[:uv_offset]
        uv = img_yuv[uv_offset:].reshape(2, -1).ravel('F')
        img_yuv = np.hstack((y, uv))


        cnt += 1

        pc_latency = (time.monotonic()-time_q[0])*1000
        time_q = time_q[1:]
        print("%2d %4d %.3f %.3f roll %6.2f ms latency %6.2f ms + %6.2f ms + %6.2f ms = %6.2f ms" % (len(msgs), evta.idx.encodeId, evt.logMonoTime/1e9, evta.idx.timestampEof/1e6, frame_latency, process_latency, network_latency, pc_latency, process_latency+network_latency+pc_latency ), len(evta.data), sock_name)

def main(addr, cams, nvidia=False):
  procs = []
  for k, v in cams.items():
    p = multiprocessing.Process(target=decoder, args=(addr, k, None, v, nvidia))
    p.start()
    procs.append(p)

  for p in procs:
    p.join()

if __name__ == "__main__":
  parser = argparse.ArgumentParser(description="Decode video streams and broadcast on VisionIPC")
  parser.add_argument("addr", help="Address of comma three")
  parser.add_argument("--nvidia", action="store_true", help="Use nvidia instead of ffmpeg")
  parser.add_argument("--cams", default="0,1,2", help="Cameras to decode")
  args = parser.parse_args()

  all_cams = [
    ("roadEncodeData", 0),
    ("wideRoadEncodeData", 1),
    ("driverEncodeData", 2),
  ]
  cams = dict([all_cams[int(x)] for x in args.cams.split(",")])
  main(args.addr, cams, args.nvidia)
