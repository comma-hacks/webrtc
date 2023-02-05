import pyautogui
import numpy
import evdev
import keymap

pyautogui.FAILSAFE = False

class DesktopControlInterface():
    def __init__(self, resolution):
        self.resolution = resolution
        self.ui = evdev.UInput()
        self.valid_actions = ["keyboard", "click", "rightclick", "mousemove", "joystick", "paste"]

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
    
    def stop(self):
        self.ui.close()