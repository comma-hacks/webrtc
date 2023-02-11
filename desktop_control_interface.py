import numpy # for mouse position interpolation
import keymap # for mapping iOS folio keys to libevdev events
import libevdev # for all low-level mouse and keyboard events
import pyautogui # only for the high-level string "paste" support
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
        for value in keymap.iOStoLinux.values():
            self.dev.enable(value)
        self.uinput = self.dev.create_uinput_device()

    def handle_action(self, action, data):
        if action == "mousemove":
            x = numpy.interp(data["cursorPositionX"], (0, data["displayWidth"]), (0, self.resolution.width))
            y = numpy.interp(data["cursorPositionY"], (0, data["displayHeight"]), (0, self.resolution.height))
            # print(f"mousemove {x} {y}")            
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
                # use keymap.reload() here to reload your source changes when adding more keys
                osKey = keymap.iOStoLinux[data["key"]]
                press = [libevdev.InputEvent(osKey, value=data["direction"]),
                    libevdev.InputEvent(libevdev.EV_SYN.SYN_REPORT, value=0)]
                self.uinput.send_events(press)
            except KeyError:
                print(f"Unknown key: {data['key']}")
        elif action == "paste":
            pyautogui.write(data["payload"]["string"])

    def supports(self, action):
        return action in ["keyboard", "leftclickbegan", "leftclickend", "rightclickbegan", "rightclickend", "mousemove", "paste"]