#!/usr/bin/env python3
"""Writes Windows resource objects (.syso) with an icon, version info and manifest.
Go links *_windows_<arch>.syso files automatically. Usage: mksyso.py <png dir> <out dir>"""
import struct, sys, os

def ico_images(d):
    out = []
    for s in [16, 24, 32, 48, 64, 128, 256]:
        b = open(os.path.join(d, f'{s}.png'), 'rb').read()
        out.append((s, b))
    return out

MANIFEST = b'''<?xml version="1.0" encoding="UTF-8" standalone="yes"?>
<assembly xmlns="urn:schemas-microsoft-com:asm.v1" manifestVersion="1.0">
  <assemblyIdentity type="win32" name="Makie" version="0.9.0.0"/>
  <description>Makie 3D part designer by nrmluvdev</description>
  <trustInfo xmlns="urn:schemas-microsoft-com:asm.v3"><security><requestedPrivileges>
    <requestedExecutionLevel level="asInvoker" uiAccess="false"/></requestedPrivileges></security></trustInfo>
  <application xmlns="urn:schemas-microsoft-com:asm.v3"><windowsSettings>
    <dpiAware xmlns="http://schemas.microsoft.com/SMI/2005/WindowsSettings">true/pm</dpiAware>
    <dpiAwareness xmlns="http://schemas.microsoft.com/SMI/2016/WindowsSettings">PerMonitorV2</dpiAwareness>
  </windowsSettings></application>
</assembly>
'''

def u16z(s): return s.encode('utf-16-le') + b'\0\0'

def vs_block(key, value=None, children=b'', vtype=1):
    """VS_VERSIONINFO-style block: wLength, wValueLength, wType, szKey, pad, value, pad, children"""
    k = u16z(key)
    hdr_len = 6 + len(k)
    pad1 = (4 - hdr_len % 4) % 4
    if value is None:
        val, vlen = b'', 0
    elif vtype == 1:
        val = u16z(value); vlen = len(val) // 2
    else:
        val = value; vlen = len(val)
    body = val
    pad2 = (4 - (hdr_len + pad1 + len(body)) % 4) % 4 if children else 0
    total = hdr_len + pad1 + len(body) + pad2 + len(children)
    return struct.pack('<HHH', total, vlen, vtype) + k + b'\0' * pad1 + body + b'\0' * pad2 + children

def align4(b): return b + b'\0' * ((4 - len(b) % 4) % 4)

def version_info():
    ffi = struct.pack('<13I', 0xFEEF04BD, 0x00010000, 0x00010000, 0, 0x00010000, 0, 0x3F, 0, 0x40004, 1, 0, 0, 0)
    strs = [('CompanyName', 'nrmluvdev'), ('LegalCopyright', '© nrmluvdev'), ('FileDescription', 'Makie 3D part designer by nrmluvdev'), ('FileVersion', '1.0.0'),
            ('InternalName', 'Makie'), ('OriginalFilename', 'Makie.exe'), ('ProductName', 'Makie'), ('ProductVersion', '0.9.0-alpha')]
    st = b''.join(align4(vs_block(k, v)) for k, v in strs)
    table = vs_block('040904B0', None, st)
    sfi = vs_block('StringFileInfo', None, align4(table))
    var = vs_block('Translation', struct.pack('<HH', 0x0409, 0x04B0), vtype=0)
    vfi = vs_block('VarFileInfo', None, align4(var))
    root = vs_block('VS_VERSION_INFO', ffi, align4(sfi) + align4(vfi), vtype=0)
    return root

def build(images, machine, reltype):
    # resources: (type, id, data)
    res = []
    grp = struct.pack('<HHH', 0, 1, len(images))
    for i, (s, png) in enumerate(images):
        res.append((3, i + 1, png))
        w = 0 if s >= 256 else s
        grp += struct.pack('<BBBBHHIH', w, w, 0, 0, 1, 32, len(png), i + 1)
    res.append((14, 1, grp))
    res.append((16, 1, version_info()))
    res.append((24, 1, MANIFEST))
    types = sorted(set(t for t, _, _ in res))
    # layout: root dir, type dirs, name dirs, lang dirs, data entries, data
    def dirhdr(n): return struct.pack('<IIHHHH', 0, 0, 0, 0, 0, n)
    root_size = 16 + 8 * len(types)
    off = root_size
    type_dirs = {}
    for t in types:
        items = [r for r in res if r[0] == t]
        type_dirs[t] = (off, items)
        off += 16 + 8 * len(items)
    name_dirs = {}
    for t in types:
        for r in type_dirs[t][1]:
            name_dirs[(t, r[1])] = off
            off += 16 + 8
    data_entries = {}
    for t in types:
        for r in type_dirs[t][1]:
            data_entries[(t, r[1])] = off
            off += 16
    data_off = {}
    for t in types:
        for r in type_dirs[t][1]:
            off = (off + 7) & ~7
            data_off[(t, r[1])] = off
            off += len(r[2])
    total = (off + 7) & ~7
    buf = bytearray(total)
    def put(o, b): buf[o:o + len(b)] = b
    put(0, dirhdr(len(types)))
    for i, t in enumerate(types):
        put(16 + 8 * i, struct.pack('<II', t, 0x80000000 | type_dirs[t][0]))
    relocs = []
    for t in types:
        o, items = type_dirs[t]
        put(o, dirhdr(len(items)))
        for i, r in enumerate(items):
            put(o + 16 + 8 * i, struct.pack('<II', r[1], 0x80000000 | name_dirs[(t, r[1])]))
            no = name_dirs[(t, r[1])]
            put(no, dirhdr(1))
            put(no + 16, struct.pack('<II', 0x0409, data_entries[(t, r[1])]))
            de = data_entries[(t, r[1])]
            put(de, struct.pack('<IIII', data_off[(t, r[1])], len(r[2]), 0, 0))
            relocs.append(de)
            put(data_off[(t, r[1])], r[2])
    raw_ptr = 20 + 40
    reloc_ptr = raw_ptr + total
    sym_ptr = reloc_ptr + 10 * len(relocs)
    hdr = struct.pack('<HHIIIHH', machine, 1, 0, sym_ptr, 1, 0, 0)
    sec = struct.pack('<8sIIIIIIHHI', b'.rsrc', 0, 0, total, raw_ptr, reloc_ptr, 0, len(relocs), 0, 0x40000040)
    rel = b''.join(struct.pack('<IIH', r, 0, reltype) for r in relocs)
    sym = struct.pack('<8sIhHBB', b'.rsrc', 0, 1, 0, 3, 0)
    strtab = struct.pack('<I', 4)
    return hdr + sec + bytes(buf) + rel + sym + strtab

imgs = ico_images(sys.argv[1])
out = sys.argv[2]
open(os.path.join(out, 'rsrc_windows_amd64.syso'), 'wb').write(build(imgs, 0x8664, 3))
open(os.path.join(out, 'rsrc_windows_arm64.syso'), 'wb').write(build(imgs, 0xAA64, 2))
open(os.path.join(out, 'rsrc_windows_386.syso'), 'wb').write(build(imgs, 0x14C, 7))
# also an .ico for convenience
ico = struct.pack('<HHH', 0, 1, len(imgs)); off = 6 + 16 * len(imgs); data = b''
for s, png in imgs:
    w = 0 if s >= 256 else s
    ico += struct.pack('<BBBBHHII', w, w, 0, 0, 1, 32, len(png), off + len(data)); data += png
open(os.path.join(out, 'web', 'favicon.ico'), 'wb').write(ico + data)
print('ok')
