# 3. On-Demand External Media Viewer

## Context

Rendering inline images and video in a terminal requires terminal-specific graphics protocols (Kitty graphics, Sixel, iTerm2 inline images). These protocols suffer from inconsistent support across terminals, multiplexers (tmux), and SSH sessions, and they distort multiline Bubbletea viewport scrolling.

## Decision

We represent media attachments in the message stream as structured metadata badges (displaying media type, filename, dimensions, and size) alongside text captions. Pressing `Enter` on a media message downloads the asset on demand to an XDG cache directory (`~/.cache/cligram/media/`) and launches it using the host operating system's default viewer (`xdg-open` on Linux, `open` on macOS).

## Consequences

- Full media support across photos, videos, audio, voice notes, and documents without terminal emulator incompatibilities.
- Zero extra dependencies on native image rendering libraries in the Pure-Go core.
- Bandwidth and disk space are conserved by downloading media exclusively on demand.
