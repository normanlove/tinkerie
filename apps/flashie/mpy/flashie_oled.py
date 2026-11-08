# flashie_oled.py - SSD1306 OLED driver (128x64 / 128x32, I2C).
# Part of Flashie by nrmluvdev. MIT licence. Based on the classic
# MicroPython ssd1306 driver design.

import framebuf


class OLED(framebuf.FrameBuffer):
    def __init__(self, i2c, width=128, height=64, addr=0x3C):
        self.i2c = i2c
        self.addr = addr
        self.width = width
        self.height = height
        self.pages = height // 8
        self.buffer = bytearray(self.pages * width)
        super().__init__(self.buffer, width, height, framebuf.MONO_VLSB)
        for c in (0xAE, 0x20, 0x00, 0x40, 0xA1, 0xA8, height - 1, 0xC8, 0xD3, 0x00,
                  0xDA, 0x02 if height == 32 else 0x12, 0xD5, 0x80, 0xD9, 0xF1,
                  0xDB, 0x30, 0x81, 0xFF, 0xA4, 0xA6, 0x8D, 0x14, 0xAF):
            self.cmd(c)
        self.fill(0)
        self.show()

    def cmd(self, c):
        self.i2c.writeto(self.addr, bytes([0x80, c]))

    def show(self):
        """Send everything drawn so far to the screen."""
        for c in (0x21, 0, self.width - 1, 0x22, 0, self.pages - 1):
            self.cmd(c)
        self.i2c.writeto(self.addr, b"\x40" + self.buffer)
