# Builds a Windows resource .syso (icon + version info) for the Go linker.
# Run from the source folder after tools/icons.py has rendered ico/*.png (needs Pillow).
import struct, io
from PIL import Image

SIZES=[16,20,24,32,40,48,64,128,256]

def dib(png):
    im=Image.open(io.BytesIO(png)).convert('RGBA'); w,h=im.size
    hdr=struct.pack('<IiiHHIIiiII',40,w,h*2,1,32,0,0,0,0,0,0)
    px=im.tobytes()
    rows=[]
    for y in range(h-1,-1,-1):
        r=px[y*w*4:(y+1)*w*4]
        b=bytearray()
        for x in range(w):
            R,G,B,A=r[x*4:x*4+4]; b+=bytes((B,G,R,A))
        rows.append(bytes(b))
    mask_row=((w+31)//32)*4
    return hdr+b''.join(rows)+b'\0'*(mask_row*h)

icons=[]
for s in SIZES:
    png=open(f'ico/{s}.png','rb').read()
    icons.append((s, png if s>=256 else dib(png)))

grp=struct.pack('<HHH',0,1,len(icons))
for i,(s,d) in enumerate(icons):
    grp+=struct.pack('<BBBBHHIH', s if s<256 else 0, s if s<256 else 0,0,0,1,32,len(d),i+1)

def u16(s): return s.encode('utf-16-le')+b'\0\0'
def pad4(b): return b+b'\0'*((4-len(b)%4)%4)
def node(key,value=b'',vlen=0,wtype=1,children=()):
    body=struct.pack('<HHH',0,vlen,wtype)+u16(key)
    body=pad4(body)+value
    if children:
        body=pad4(body)
        for i,c in enumerate(children):
            body+= c if i==len(children)-1 else pad4(c)
    return struct.pack('<H',len(body))+body[2:]
def sstr(k,v): return node(k,u16(v),len(v)+1,1)
ver=(1,0,0,0)
ms=(ver[0]<<16)|ver[1]; ls=(ver[2]<<16)|ver[3]
fixed=struct.pack('<13I',0xFEEF04BD,0x00010000,ms,ls,ms,ls,0x3F,0,0x40004,1,0,0,0)
strings=[('CompanyName','nrmluvdev'),('FileDescription','Tracie - serial monitor + logic analyser'),
 ('FileVersion','0.5.0-alpha'),('InternalName','Tracie'),('LegalCopyright','(c) 2026 nrmluvdev - free to use'),
 ('OriginalFilename','Tracie.exe'),('ProductName','Tracie (nrmluvdev hobbyist suite)'),('ProductVersion','0.5.0-alpha'),('Comments','Made by nrmluvdev')]
st=node('040904B0',b'',0,1,[sstr(k,v) for k,v in strings])
sfi=node('StringFileInfo',b'',0,1,[st])
var=node('Translation',struct.pack('<HH',0x0409,0x04B0),4,0)
vfi=node('VarFileInfo',b'',0,1,[var])
version=node('VS_VERSION_INFO',fixed,len(fixed),0,[sfi,vfi])

# resource tree: type -> [(id, data)]
tree=[(3,[(i+1,d) for i,(s,d) in enumerate(icons)]),(14,[(1,grp)]),(16,[(1,version)])]
def dirhdr(n): return struct.pack('<IIHHHH',0,0,0,0,0,n)
# layout
off=16+8*len(tree)
type_off=[]
for t,names in tree:
    type_off.append(off); off+=16+8*len(names)
name_off=[]
for t,names in tree:
    l=[]
    for n,_ in names: l.append(off); off+=16+8
    name_off.append(l)
data_entry_off=[]
for t,names in tree:
    l=[]
    for n,_ in names: l.append(off); off+=16
    data_entry_off.append(l)
blobs=bytearray(); blob_off=[]
base=off
for t,names in tree:
    l=[]
    for n,d in names:
        while (base+len(blobs))%8: blobs+=b'\0'
        l.append(base+len(blobs)); blobs+=d
    blob_off.append(l)
sec=bytearray(dirhdr(len(tree)))
for i,(t,names) in enumerate(tree): sec+=struct.pack('<II',t,0x80000000|type_off[i])
for i,(t,names) in enumerate(tree):
    sec+=dirhdr(len(names))
    for j,(n,_) in enumerate(names): sec+=struct.pack('<II',n,0x80000000|name_off[i][j])
for i,(t,names) in enumerate(tree):
    for j,_ in enumerate(names):
        sec+=dirhdr(1)+struct.pack('<II',0x0409,data_entry_off[i][j])
relocs=[]
for i,(t,names) in enumerate(tree):
    for j,(n,d) in enumerate(names):
        relocs.append(len(sec))
        sec+=struct.pack('<IIII',blob_off[i][j],len(d),0,0)
assert len(sec)==base
sec+=blobs
while len(sec)%4: sec+=b'\0'
raw_ptr=20+40
reloc_ptr=raw_ptr+len(sec)
sym_ptr=reloc_ptr+10*len(relocs)
out=bytearray(struct.pack('<HHIIIHH',0x8664,1,0,sym_ptr,1,0,0))
out+=struct.pack('<8sIIIIIIHHI',b'.rsrc',0,0,len(sec),raw_ptr,reloc_ptr,0,len(relocs),0,0x40000040)
out+=sec
for r in relocs: out+=struct.pack('<IIH',r,0,3)
out+=struct.pack('<8sIhHBB',b'.rsrc',0,1,0,3,0)
out+=struct.pack('<I',4)
open('rsrc_windows_amd64.syso','wb').write(out)
print(len(out),'bytes')
