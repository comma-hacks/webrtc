import numpy
import keymap
import libevdev
from libevdev import InputEvent, InputAbsInfo

class DesktopControlInterface():
    def __init__(self, resolution):
        global uinput
        self.resolution = resolution
        self.dev = libevdev.Device()
        self.dev.name = 'virtual mouse and keyboard'
        x_info = InputAbsInfo(minimum=0, maximum=resolution.width, resolution=200)
        self.dev.enable(libevdev.EV_ABS.ABS_X, data=x_info)
        y_info = InputAbsInfo(minimum=0, maximum=resolution.height, resolution=200)
        self.dev.enable(libevdev.EV_ABS.ABS_Y, data=y_info)
        self.dev.enable(libevdev.EV_KEY.BTN_LEFT)
        self.dev.enable(libevdev.EV_KEY.BTN_MIDDLE)
        self.dev.enable(libevdev.EV_KEY.BTN_RIGHT)
        self.uinput = self.dev.create_uinput_device()

    def handle_action(self, action, data):
        if action == "mousemove":
            x = numpy.interp(data["cursorPositionX"], (0, data["displayWidth"]), (0, self.resolution.width))
            y = numpy.interp(data["cursorPositionY"], (0, data["displayHeight"]), (0, self.resolution.height))
            print(f"mousemove {x} {y}")            
            events = [InputEvent(libevdev.EV_ABS.ABS_X, int(x)),
                    InputEvent(libevdev.EV_ABS.ABS_Y, int(y)),
                    InputEvent(libevdev.EV_SYN.SYN_REPORT, 0)]
            self.uinput.send_events(events)

        elif action == "leftclickbegan":
            press = [libevdev.InputEvent(libevdev.EV_KEY.BTN_LEFT, value=1),
                    libevdev.InputEvent(libevdev.EV_SYN.SYN_REPORT, value=0)]
            self.uinput.send_events(press)
        elif action == "leftclickend":
            press = [libevdev.InputEvent(libevdev.EV_KEY.BTN_LEFT, value=0),
                    libevdev.InputEvent(libevdev.EV_SYN.SYN_REPORT, value=0)]
            self.uinput.send_events(press)
        elif action == "rightclickbegan":
            press = [libevdev.InputEvent(libevdev.EV_KEY.BTN_RIGHT, value=1),
                    libevdev.InputEvent(libevdev.EV_SYN.SYN_REPORT, value=0)]
            self.uinput.send_events(press)
        elif action == "rightclickend":
            press = [libevdev.InputEvent(libevdev.EV_KEY.BTN_RIGHT, value=0),
                    libevdev.InputEvent(libevdev.EV_SYN.SYN_REPORT, value=0)]
            self.uinput.send_events(press)
        elif action == "keyboard":
            try:
                # keymap.reload()
                osKey = keymap.iOStoLinux[data["key"]]
                print(osKey)
                # self.ui.write(e.EV_KEY, osKey, data["direction"])
                # self.ui.syn()
            except KeyError:
                print(f"Unknown key: {data['key']}")
        elif action == "paste":
            pass
            # might as well support the secureput protocol completely
            # pyautogui.write(data["payload"]["string"])

    def supports(self, action):
        return action in ["keyboard", "leftclickbegan", "leftclickend", "rightclickbegan", "rightclickend", "mousemove", "paste"]

    def stop(self):
        pass
        # self.ui.close()


if __name__ == "__main__":
    import time
    import libevdev
    from libevdev import InputEvent, InputAbsInfo
    dev = libevdev.Device()
    dev.name = 'some test device'
    a = InputAbsInfo(minimum=0, maximum=1920, resolution=200)
    dev.enable(libevdev.EV_ABS.ABS_X, data=a)
    a = InputAbsInfo(minimum=0, maximum=1080, resolution=200)
    dev.enable(libevdev.EV_ABS.ABS_Y, data=a)
    dev.enable(libevdev.EV_KEY.BTN_LEFT)
    dev.enable(libevdev.EV_KEY.BTN_MIDDLE)
    dev.enable(libevdev.EV_KEY.BTN_RIGHT)

    uinput = dev.create_uinput_device()
    print("New device at {} ({})".format(uinput.devnode, uinput.syspath))

    # Sleep for a bit so udev, libinput, Xorg, Wayland, ... all have had
    # a chance to see the device and initialize it. Otherwise the event
    # will be sent by the kernel but nothing is ready to listen to the
    # device yet.
    time.sleep(1)


    events = [InputEvent(libevdev.EV_ABS.ABS_X, 1900),
            InputEvent(libevdev.EV_ABS.ABS_Y, 1000),
            InputEvent(libevdev.EV_SYN.SYN_REPORT, 0)]
