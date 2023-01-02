#!/bin/bash
docker run --rm -v $PWD/webrtc:/project/webrtc -w /project/webrtc -p 8080:8080 -it webrtc-body python3 server.py 192.168.99.200