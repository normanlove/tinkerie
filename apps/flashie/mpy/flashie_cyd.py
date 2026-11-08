# flashie_cyd.py - a small display + touch driver for Cheap Yellow Displays
# (and other SPI TFTs: ILI9341, ST7789, ST7796).
# Part of Flashie by nrmluvdev. MIT licence. Uploaded to your board by
# Flashie's Firmware Creator when you use the screen blocks.

from machine import Pin, SPI, SoftSPI, SoftI2C
import framebuf
import time


def rgb(r, g, b):
    """Turn red/green/blue (0-255 each) into the 16-bit colour the screen uses."""
    return ((r & 0xF8) << 8) | ((g & 0xFC) << 3) | (b >> 3)


def _swap(c):
    # framebuf stores RGB565 little-endian; the screen wants big-endian.
    return ((c & 0xFF) << 8) | ((c >> 8) & 0xFF)


class Screen:
    def __init__(self, w, h, sck, mosi, cs, dc, bl=-1, miso=-1, rst=-1,
                 rot=1, bgr=True, invert=False, xoff=0, yoff=0,
                 spi_id=1, baud=40000000, touch=None):
        kw = {"baudrate": baud, "sck": Pin(sck), "mosi": Pin(mosi)}
        if miso >= 0:
            kw["miso"] = Pin(miso)
        self.spi = SPI(spi_id, **kw)
        self.baud = baud
        self.cs = Pin(cs, Pin.OUT, value=1)
        self.dc = Pin(dc, Pin.OUT, value=0)
        if rst >= 0:
            r = Pin(rst, Pin.OUT, value=1)
            r(0)
            time.sleep_ms(20)
            r(1)
            time.sleep_ms(120)
        self._w0, self._h0 = w, h
        self.xoff, self.yoff = xoff, yoff
        self.cmd(0x01)                 # software reset
        time.sleep_ms(150)
        self.cmd(0x11)                 # wake up
        time.sleep_ms(120)
        self.cmd(0x3A, b"\x55")        # 16 bits per pixel
        self.cmd(0x21 if invert else 0x20)
        self.bgr = bgr
        self.rotation(rot)
        self.cmd(0x29)                 # display on
        time.sleep_ms(20)
        if bl >= 0:
            self.backlight = Pin(bl, Pin.OUT, value=1)
        self.tx = self.ty = 0
        self._touch = None
        if touch:
            kind, cfg = touch
            if kind == "xpt":
                self._touch = _XPT2046(self, **cfg)
            elif kind == "gt911":
                self._touch = _GT911(self, **cfg)
            elif kind == "cst":
                self._touch = _CST816(self, **cfg)

    # --- low level -------------------------------------------------------
    def cmd(self, c, data=None):
        self.cs(0)
        self.dc(0)
        self.spi.write(bytes([c]))
        if data:
            self.dc(1)
            self.spi.write(data)
        self.cs(1)

    def rotation(self, r):
        mad = (0x48, 0x28, 0x88, 0xE8)[r & 3]
        if not self.bgr:
            mad &= ~0x08
        self.cmd(0x36, bytes([mad]))
        self.rot = r & 3
        if self.rot % 2 == 0:
            self.width, self.height = self._w0, self._h0
        else:
            self.width, self.height = self._h0, self._w0

    def _window(self, x, y, w, h):
        if self.rot % 2 == 0:
            x += self.xoff
            y += self.yoff
        else:
            x += self.yoff
            y += self.xoff
        self.cmd(0x2A, bytes([x >> 8, x & 0xFF, (x + w - 1) >> 8, (x + w - 1) & 0xFF]))
        self.cmd(0x2B, bytes([y >> 8, y & 0xFF, (y + h - 1) >> 8, (y + h - 1) & 0xFF]))
        self.cs(0)
        self.dc(0)
        self.spi.write(b"\x2C")
        self.dc(1)

    def blit(self, buf, x, y, w, h):
        """Send raw RGB565 pixels (big-endian) to a rectangle."""
        self._window(x, y, w, h)
        self.spi.write(buf)
        self.cs(1)

    # --- drawing ---------------------------------------------------------
    def rect(self, x, y, w, h, color):
        """Filled rectangle. Parts outside the screen are cut off."""
        x, y, w, h = int(x), int(y), int(w), int(h)
        if x < 0:
            w += x
            x = 0
        if y < 0:
            h += y
            y = 0
        w = min(w, self.width - x)
        h = min(h, self.height - y)
        if w <= 0 or h <= 0:
            return
        line = bytes([color >> 8, color & 0xFF]) * w
        self._window(x, y, w, h)
        for _ in range(h):
            self.spi.write(line)
        self.cs(1)

    def fill(self, color):
        self.rect(0, 0, self.width, self.height, color)

    def pixel(self, x, y, color):
        self.rect(x, y, 1, 1, color)

    def text(self, s, x, y, size=1, color=0xFFFF, bg=0x0000):
        """Write text with the built-in 8x8 font, scaled up `size` times."""
        s = str(s)
        x, y, size = int(x), int(y), max(1, int(size))
        w = 8 * len(s)
        if w == 0:
            return
        buf = bytearray(w * 16)
        fb = framebuf.FrameBuffer(buf, w, 8, framebuf.RGB565)
        fb.fill(_swap(bg))
        fb.text(s, 0, 0, _swap(color))
        if size == 1:
            self.blit(buf, x, y, w, 8)
            return
        mv = memoryview(buf)
        row = bytearray(w * size * 2)
        for r in range(8):
            o = 0
            for c in range(w):
                px = mv[(r * w + c) * 2:(r * w + c) * 2 + 2]
                for _ in range(size):
                    row[o] = px[0]
                    row[o + 1] = px[1]
                    o += 2
            for k in range(size):
                if y + r * size + k < self.height:
                    self.blit(row, x, y + r * size + k, min(w * size, self.width - x), 1)

    # --- touch -----------------------------------------------------------
    def touched(self):
        """True while the screen is being pressed. Updates .tx / .ty."""
        if not self._touch:
            return False
        p = self._touch.read()
        if p:
            self.tx, self.ty = p
            return True
        return False


class _XPT2046:
    # Resistive touch. Calibrated for the CYD in landscape (rotation 1).
    def __init__(self, scr, cs, irq=-1, sck=-1, mosi=-1, miso=-1, shared=False,
                 xmin=200, xmax=3700, ymin=240, ymax=3800):
        self.scr = scr
        self.cs = Pin(cs, Pin.OUT, value=1)
        self.irq = Pin(irq, Pin.IN) if irq >= 0 else None
        self.shared = shared
        if shared:
            self.spi = scr.spi
        else:
            self.spi = SoftSPI(baudrate=1000000, sck=Pin(sck), mosi=Pin(mosi), miso=Pin(miso))
        self.cal = (xmin, xmax, ymin, ymax)
        self.buf = bytearray(3)

    def _read(self, cmd):
        self.spi.write_readinto(bytes([cmd, 0, 0]), self.buf)
        return ((self.buf[1] << 8) | self.buf[2]) >> 3

    def read(self):
        if self.irq and self.irq.value():
            return None
        if self.shared:
            self.spi.init(baudrate=2000000)
        self.cs(0)
        z = self._read(0xB1) + 4095 - self._read(0xC1)
        p = None
        if z > 400:
            # Same command order as the XPT2046_Touchscreen library the CYD
            # examples use: 0x91 = X, 0xD1 = Y in landscape.
            xs = [self._read(0x91) for _ in range(3)]
            ys = [self._read(0xD1) for _ in range(3)]
            rx, ry = sorted(xs)[1], sorted(ys)[1]
            xmin, xmax, ymin, ymax = self.cal
            # position in landscape (rotation 1) first
            lx = (rx - xmin) * 320 // (xmax - xmin)
            ly = (ry - ymin) * 240 // (ymax - ymin)
            s = self.scr
            if s.rot == 1:
                x, y = lx, ly
            elif s.rot == 3:
                x, y = s.width - lx, s.height - ly
            elif s.rot == 0:
                x, y = s.width - ly * s.width // 240, lx * s.height // 320
            else:
                x, y = ly * s.width // 240, s.height - lx * s.height // 320
            p = (max(0, min(s.width - 1, x)), max(0, min(s.height - 1, y)))
        self.cs(1)
        if self.shared:
            self.spi.init(baudrate=self.scr.baud)
        return p


class _GT911:
    # Capacitive touch (3.2" and 3.5" "C" CYDs).
    def __init__(self, scr, sda, scl, rst=-1, irq=-1, addr=0x5D):
        self.scr = scr
        self.i2c = SoftI2C(sda=Pin(sda), scl=Pin(scl), freq=400000)
        self.addr = addr
        found = self.i2c.scan()
        if addr not in found and 0x14 in found:
            self.addr = 0x14

    def read(self):
        try:
            st = self.i2c.readfrom_mem(self.addr, 0x814E, 1, addrsize=16)[0]
            if not (st & 0x80):
                return None
            p = None
            if st & 0x0F:
                d = self.i2c.readfrom_mem(self.addr, 0x8150, 4, addrsize=16)
                p = self._map(d[0] | (d[1] << 8), d[2] | (d[3] << 8))
            self.i2c.writeto_mem(self.addr, 0x814E, b"\x00", addrsize=16)
            return p
        except OSError:
            return None

    def _map(self, x, y):
        s = self.scr
        w0, h0 = s._w0, s._h0
        if s.rot == 0:
            return x, y
        if s.rot == 1:
            return y, w0 - 1 - x
        if s.rot == 2:
            return w0 - 1 - x, h0 - 1 - y
        return h0 - 1 - y, x


class _CST816(_GT911):
    # Capacitive touch on the 2.4" "C" CYD (CST816 / CST820).
    def __init__(self, scr, sda, scl, rst=-1, irq=-1, addr=0x15):
        self.scr = scr
        self.i2c = SoftI2C(sda=Pin(sda), scl=Pin(scl), freq=400000)
        self.addr = addr
        if rst >= 0:
            r = Pin(rst, Pin.OUT, value=0)
            time.sleep_ms(10)
            r(1)
            time.sleep_ms(50)

    def read(self):
        try:
            d = self.i2c.readfrom_mem(self.addr, 0x02, 5)
        except OSError:
            return None
        if not (d[0] & 0x0F):
            return None
        return self._map(((d[1] & 0x0F) << 8) | d[2], ((d[3] & 0x0F) << 8) | d[4])
