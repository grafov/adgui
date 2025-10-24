# adgui

Simple GUI for control CLI for AdGuard VPN. Linux XLibre/Wayland.

**Works partially yet. Development just in progress.**

## Installation

It tries to install into protected directory `/usr/local/bin` that requires root privileges.
Use "sudo", "doas" or other appropriate command with SUDO environvent dir:

`SUDO=sudo make install`

Or use `PREFIX` for installing to another directory, for example under home:

`PREFIX=~/bin make install`

