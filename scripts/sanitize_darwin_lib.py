#!/usr/bin/env python3
"""
sanitize_darwin_lib.py

Scans a Mach-O static library (.a) for ntgcalls.cpp.o object members.
In those members, neutralizes the __LD,__compact_unwind section by renaming it
to __compact_unused and clearing its size and relocation count.

This resolves the Apple classic linker (ld_classic) failure:
  ld: in .../libntgcalls.a(ntgcalls.cpp.o), too many compact unwind infos in function ...
which is caused by >70 compact unwind entries in a single AsyncPromise atom exceeding
ld_classic's internal per-function atom table limit.
"""

import os
import struct
import sys


def patch_macho_bytes(data: bytearray, member_name: str) -> bool:
    if len(data) < 32:
        return False
    magic, = struct.unpack('<I', data[:4])
    if magic != 0xfeedfacf:  # MH_MAGIC_64
        return False

    _, _, _, ncmds, _, _, _ = struct.unpack('<IIIIIII', data[4:32])
    offset = 32
    patched = False
    for _ in range(ncmds):
        if offset + 8 > len(data):
            break
        cmd, cmdsize = struct.unpack('<II', data[offset:offset+8])
        if cmd == 0x19:  # LC_SEGMENT_64
            nsects, = struct.unpack('<I', data[offset+64:offset+68])
            sect_offset = offset + 72
            for _ in range(nsects):
                if sect_offset + 80 > len(data):
                    break
                sectname = bytes(data[sect_offset:sect_offset+16]).rstrip(b'\x00')
                if sectname == b'__compact_unwind':
                    # Rename section so ld64 compact unwind pass skips it without invalidating section address range
                    data[sect_offset:sect_offset+16] = b'__compact_unused'
                    patched = True
                sect_offset += 80
        offset += cmdsize
    return patched


def patch_archive(archive_path: str):
    if not os.path.exists(archive_path):
        print(f"Archive not found: {archive_path}", file=sys.stderr)
        return

    with open(archive_path, 'r+b') as f:
        magic = f.read(8)
        if magic != b'!<arch>\n':
            f.seek(0)
            data = bytearray(f.read())
            if patch_macho_bytes(data, os.path.basename(archive_path)):
                f.seek(0)
                f.write(data)
                print(f"--> [sanitize] Patched raw Mach-O object {archive_path}")
            return

        patched_count = 0
        while True:
            hdr = f.read(60)
            if len(hdr) < 60:
                break
            name = hdr[:16].decode('ascii', errors='replace').strip()
            size = int(hdr[48:58].strip())
            hdr_end = f.tell()
            namelen = 0
            if name.startswith('#1/'):
                namelen = int(name[3:])
                realname = f.read(namelen).decode('ascii', errors='replace').rstrip('\x00')
            else:
                realname = name

            member_data_offset = f.tell()
            member_data = bytearray(f.read(size - namelen))
            if patch_macho_bytes(member_data, realname):
                f.seek(member_data_offset)
                f.write(member_data)
                patched_count += 1

            f.seek(hdr_end + size + (size % 2))

        print(f"--> [sanitize] Successfully neutralized __compact_unwind in {patched_count} object(s) in {archive_path}")


if __name__ == '__main__':
    if len(sys.argv) < 2:
        print("Usage: sanitize_darwin_lib.py <path_to_libntgcalls.a>", file=sys.stderr)
        sys.exit(1)
    patch_archive(sys.argv[1])
