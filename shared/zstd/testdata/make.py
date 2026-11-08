# Makes shared/zstd/testdata with Python 3.14's compression.zstd (libzstd).
import sys, os, random, hashlib
from compression import zstd
out = sys.argv[1]; os.makedirs(out, exist_ok=True)
r = random.Random(7)
base = r.randbytes(256<<10)
filler = b"".join([(b"Tinkerie by nrmluvdev %d\n" % i) * 2000 + b"\0" * (300<<10) + r.randbytes(4<<10) for i in range(36)])
data = base + filler + base + r.randbytes(64<<10) + b"\0"*(1<<20)   # base repeats > 8 MiB back
assert len(filler) > 9<<20
open(f"{out}/plain.sha256","w").write(hashlib.sha256(data).hexdigest() + " %d\n" % len(data))
P = zstd.CompressionParameter
def put(n, b): open(f"{out}/{n}.zst","wb").write(b)
put("level19", zstd.compress(data, options={P.compression_level:19, P.checksum_flag:1}))
put("long27", zstd.compress(data, options={P.compression_level:9, P.enable_long_distance_matching:1, P.window_log:27, P.checksum_flag:1}))
c = zstd.ZstdCompressor(level=3)   # streamed: no size in the header
put("nosize", c.compress(data[:len(data)//2]) + c.compress(data[len(data)//2:]) + c.flush())
skip = (0x184D2A50).to_bytes(4,"little") + (5).to_bytes(4,"little") + b"hello"
put("multi", zstd.compress(data[:3<<20]) + skip + zstd.compress(data[3<<20:], options={P.checksum_flag:1}))
d = bytearray(zstd.compress(data, options={P.compression_level:3, P.checksum_flag:1}))
d[len(d)*2//3] ^= 0x40
put("damaged", bytes(d))
put("truncated", bytes(d[:len(d)//3]))
c = zstd.ZstdCompressor(options={P.compression_level:1, P.window_log:31})
put("hugewindow", c.compress(b"x"*1000) + c.flush())
print(len(data))
