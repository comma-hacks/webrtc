from evdev import ecodes as e

# Some info can be found here
# https://www.kernel.org/doc/Documentation/input/event-codes.txt

# The full list of key codes for linux can be found here:
# https://github.com/torvalds/linux/blob/master/include/uapi/linux/input-event-codes.h

# I could not find a good list for iOS so I just tapped the keys one by one.
# I only have access to the iPad Folio keyboard so those are the only keys mapped.

# Also, my caps lock is mapped to escape in the iOS settings, so no caps lock is mapped.
# This keyboard lacks a physical escape key (but the capslock map is a good consolation)

# Map of iOS to Linux key codes
iOStoLinux = {
  # Row 1 of 5
  53: e.KEY_GRAVE,
  30: e.KEY_1,
  31: e.KEY_2,
  32: e.KEY_3,
  33: e.KEY_4,
  34: e.KEY_5,
  35: e.KEY_6,
  36: e.KEY_7,
  37: e.KEY_8,
  38: e.KEY_9,
  39: e.KEY_0,
  45: e.KEY_MINUS,
  46: e.KEY_EQUAL,
  42: e.KEY_BACKSPACE,

  # Row 2 of 5
  43: e.KEY_TAB,
  20: e.KEY_Q,
  26: e.KEY_W,
  8: e.KEY_E,
  21: e.KEY_R,
  23: e.KEY_T,
  28: e.KEY_Y,
  24: e.KEY_U,
  12: e.KEY_I,
  18: e.KEY_O,
  19: e.KEY_P,
  47: e.KEY_LEFTBRACE,
  48: e.KEY_RIGHTBRACE,
  49: e.KEY_BACKSLASH,

  # Row 3 of 5
  41: e.KEY_ESC,
  4: e.KEY_A,
  22: e.KEY_S,
  7: e.KEY_D,
  9: e.KEY_F,
  10: e.KEY_G,
  11: e.KEY_H,
  13: e.KEY_J,
  14: e.KEY_K,
  15: e.KEY_L,
  51: e.KEY_SEMICOLON,
  52: e.KEY_APOSTROPHE,
  40: e.KEY_ENTER,

  # Row 4 of 5
  225: e.KEY_LEFTSHIFT,
  29: e.KEY_Z,
  27: e.KEY_X,
  6: e.KEY_C,
  25: e.KEY_V,
  5: e.KEY_B,
  17: e.KEY_N,
  16: e.KEY_M,
  54: e.KEY_COMMA,
  55: e.KEY_DOT,
  56: e.KEY_SLASH,
  229: e.KEY_RIGHTSHIFT,

  # Row 5 of 5
  224: e.KEY_LEFTCTRL,
  226: e.KEY_LEFTALT,
  227: e.KEY_LEFTMETA,
  44: e.KEY_SPACE,
  231: e.KEY_RIGHTMETA,
  230: e.KEY_RIGHTALT,
  80: e.KEY_LEFT,
  82: e.KEY_UP,
  81: e.KEY_DOWN,
  79: e.KEY_RIGHT
}

# Useful while creating the mapping...
# import importlib
# import sys
# def reload():
#   importlib.reload(sys.modules[__name__])