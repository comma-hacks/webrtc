sudo apt-get update
sudo apt-get upgrade -y
sudo apt install -y curl
sudo apt install -y git
sudo apt install -y build-essential
sudo apt install -y ca-certificates
sudo apt install -y autoconf
sudo apt install -y python3-pip
sudo apt install -y ffmpeg
sudo apt install -y clang
sudo apt install -y ocl-icd-opencl-dev
sudo apt install -y capnproto
sudo apt install -y libcapnp-dev
sudo apt install -y libzmq3-dev
sudo apt install -y python3-openssl




sudo apt install -y libbz2-dev
sudo apt install -y libffi-dev
sudo apt install -y liblzma-dev
sudo apt install -y libncurses5-dev
sudo apt install -y libncursesw5-dev
sudo apt install -y libreadline-dev
sudo apt install -y libsqlite3-dev
sudo apt install -y libssl-dev
sudo apt install -y libtool
sudo apt install -y llvm
sudo apt install -y make
sudo apt install -y opencl-headers 

sudo apt install -y tk-dev
sudo apt install -y wget
sudo apt install -y xz-utils
sudo apt install -y zlib1g-dev

sudo apt install python-is-python3


pip3 install pyyaml==5.1.2 Cython==0.29.14 scons==3.1.1 numpy==1.21.1 pycapnp==1.1.1

python3 -m site --user-base

export PATH="$HOME/.local/bin:$PATH"

cd cereal
scons -c && scons -j$(nproc)
cd ..



https://blog.eiler.eu/posts/20210117/
https://github.com/PyAV-Org/PyAV/issues/798

sudo apt-get install -y \
    libavformat-dev libavcodec-dev libavdevice-dev \
    libavutil-dev libswscale-dev libswresample-dev libavfilter-dev

sudo apt-get install libopus-dev libvpx-dev

pip3 install --no-binary :all: aiortc aiohttp





```
pi@raspberrypi:~/webrtc-body $ ./compressed_vipc_track.py
Warning, using python time.time() instead of faster sec_since_boot
waiting for iframe
[hevc_v4l2m2m @ 0x41157160] level=-99
[hevc_v4l2m2m @ 0x41157160] Could not find a valid device
[hevc_v4l2m2m @ 0x41157160] can't configure decoder
Traceback (most recent call last):
  File "/home/pi/webrtc-body/./compressed_vipc_track.py", line 82, in <module>
    loop.run_until_complete(test())
  File "/usr/lib/python3.9/asyncio/base_events.py", line 642, in run_until_complete
    return future.result()
  File "/home/pi/webrtc-body/./compressed_vipc_track.py", line 68, in test
    await track.recv()
  File "/home/pi/webrtc-body/./compressed_vipc_track.py", line 45, in recv
    self.codec.decode(av.packet.Packet(evta.header))
  File "av/codec/context.pyx", line 507, in av.codec.context.CodecContext.decode
  File "av/codec/context.pyx", line 519, in av.codec.context.CodecContext.decode
  File "av/codec/context.pyx", line 289, in av.codec.context.CodecContext.open
  File "av/error.pyx", line 336, in av.error.err_check
av.error.ValueError: [Errno 22] Invalid argument
```