#!/bin/bash
docker run --rm -v $PWD/webrtc:/project/webrtc -w /project/webrtc -it webrtc-body bash