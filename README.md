# Overview

This project implements a WebRTC service for the Comma 3 which means that it aims to transmit the video feeds as fast as possible.

## uinput

/etc/udev/rules.d/10-allow-uinput.rules 

```
# uncomment in case of:
#     evdev.uinput.UInputError: "/dev/uinput" cannot be opened for writing
# see https://bugs.debian.org/cgi-bin/bugreport.cgi?bug=827240
#KERNEL=="uinput", SUBSYSTEM=="misc", OPTIONS+="static_node=uinput", TAG+="uaccess", GROUP="input", MODE="0660"
```