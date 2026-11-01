#!/usr/bin/env python3
"""
sanitize_darwin_lib.py

Scans a Mach-O static library (.a) and sanitizes unwinding metadata to satisfy
the Apple classic linker (ld_classic):

1. Neutralizes __TEXT,__eh_frame sections in all member objects by renaming them
   to __DWARF,__debug_frame. Upstream pytgcalls packaging generates broken DWARF
   FDEs whose relocation targets point outside code sections (e.g. in ffmpeg/vp9/opus),
   causing ld_classic to crash with:
     ld: Assertion failed: (targetAtom != NULL), function Fixup, file ld.hpp, line 1058.
   Since all Darwin arm64 objects in libntgcalls.a also contain full __compact_unwind
   tables, moving __eh_frame to __DWARF safely suppresses DWARF CFI parsing without
   affecting unwinding.

2. Neutralizes __LD,__compact_unwind in heavily templated C++ objects (or any object
   with >15 compact unwind records for a single function atom) by renaming to
   __DWARF,__debug_unwind. This resolves ld_classic's 4-bit limit error:
     ld: in .../libntgcalls.a(ntgcalls.cpp.o), too many compact unwind infos in function ...

3. Normalizes __DATA,__objc_selrefs section type from S_LITERAL_POINTERS (0x5)
   to S_REGULAR (0x0). This resolves the Apple classic linker failure:
     ld: Assertion failed: (0 && "unsupported reference to selector"), function targetCString, file macho_relocatable_file.cpp, line 6826.
   which occurs when ld_classic attempts to parse modern 64-bit selector references
   as literal C-string pointers in its legacy ObjC1 pass.

4. Clears relocations from __DATA,__thread_vars sections.
   Upstream packaging stripped local symbols from the static library, which caused
   the third field of TLV descriptors (offset 0x10) to point to external symbol 0
   (such as ___cxa_get_globals or guard variables). In multi-descriptor objects,
   secondary descriptors also placed __tlv_bootstrap at non-zero offsets within
   single aggregated atoms. Both trigger ld_classic's assertion:
     ld: Assertion failed: (fit->binding == ld::Fixup::bindingDirectlyBound && "thread variable def contains pointer to global"), function doPass, file tlvp.cpp, line 278.
   Zeroing out relocations on __thread_vars eliminates non-zero fixup offsets in TLV atoms
   while preserving the section's S_THREAD_LOCAL_VARIABLES definition, preventing the crash.
"""

import os
import struct
import sys
from collections import defaultdict


TARGET_CU_MEMBERS = (
    'ntgcalls.cpp',
    'h264_decoder.cpp',
    'src_mc.cpp',
    'src_downsample_downsample.cpp',
)


def has_oversized_compact_unwind(data: bytearray) -> bool:
    """Returns True if any function atom in the Mach-O object has > 15 compact unwind records."""
    if len(data) < 32:
        return False
    magic, = struct.unpack('<I', data[:4])
    if magic != 0xfeedfacf:
        return False
    _, _, _, ncmds, _, _, _ = struct.unpack('<IIIIIII', data[4:32])
    offset = 32
    cu_sect = None
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
                    addr, size, file_offset, align, reloff, nreloc = struct.unpack(
                        '<QQIIII', data[sect_offset+32:sect_offset+64]
                    )
                    cu_sect = (reloff, nreloc)
                sect_offset += 80
        offset += cmdsize

    if not cu_sect:
        return False
    reloff, nreloc = cu_sect
    if nreloc <= 15:
        return False

    entry_symbols = defaultdict(int)
    for i in range(nreloc):
        if reloff + (i + 1) * 8 > len(data):
            break
        r_addr, r_info = struct.unpack('<II', data[reloff + i*8:reloff + (i+1)*8])
        r_symnum = r_info & 0xFFFFFF
        if (r_addr % 32) == 0:  # Code address relocation
            entry_symbols[r_symnum] += 1
            if entry_symbols[r_symnum] > 15:
                return True
    return False


def patch_macho_bytes(data: bytearray, member_name: str) -> bool:
    if len(data) < 32:
        return False
    magic, = struct.unpack('<I', data[:4])
    if magic != 0xfeedfacf:  # MH_MAGIC_64
        return False

    _, _, _, ncmds, _, _, _ = struct.unpack('<IIIIIII', data[4:32])
    offset = 32
    patched = False

    should_patch_cu = any(t in member_name for t in TARGET_CU_MEMBERS) or has_oversized_compact_unwind(data)

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
                if should_patch_cu and sectname == b'__compact_unwind':
                    data[sect_offset:sect_offset+16] = b'__debug_unwind\x00\x00'
                    data[sect_offset+16:sect_offset+32] = b'__DWARF\x00\x00\x00\x00\x00\x00\x00\x00\x00'
                    patched = True
                elif sectname == b'__eh_frame':
                    data[sect_offset:sect_offset+16] = b'__debug_frame\x00\x00\x00'
                    data[sect_offset+16:sect_offset+32] = b'__DWARF\x00\x00\x00\x00\x00\x00\x00\x00\x00'
                    patched = True
                elif sectname == b'__objc_selrefs':
                    flags, = struct.unpack('<I', data[sect_offset+64:sect_offset+68])
                    if (flags & 0xff) == 5:  # S_LITERAL_POINTERS -> S_REGULAR
                        struct.pack_into('<I', data, sect_offset+64, flags & ~0xff)
                        patched = True
                elif sectname == b'__thread_vars':
                    reloff, nreloc = struct.unpack('<II', data[sect_offset+56:sect_offset+64])
                    if nreloc > 0:
                        struct.pack_into('<II', data, sect_offset+56, 0, 0)
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

        print(f"--> [sanitize] Successfully sanitized {patched_count} object(s) in {archive_path}")


if __name__ == '__main__':
    if len(sys.argv) < 2:
        print("Usage: sanitize_darwin_lib.py <path_to_libntgcalls.a>", file=sys.stderr)
        sys.exit(1)
    patch_archive(sys.argv[1])
