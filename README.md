# adgui

Simple GUI for control CLI for AdGuard VPN. Linux XLibre/Wayland.

**Works partially yet. Development just in progress.**

## Installation

It tries to install into protected directory `/usr/local/bin` that requires root privileges.
Use "sudo", "doas" or other appropriate command with SUDO environvent dir:

`SUDO=sudo make install`

Or use `PREFIX` for installing to another directory, for example under home:

`PREFIX=~/bin make install`

## Features

### Site Exclusions (Domains Tab)

The Domains tab in the dashboard allows you to manage site exclusions for AdGuard VPN. You can configure which domains bypass or use the VPN connection.

#### Exclusion Modes- **General mode**: Domains in the list are excluded from VPN (traffic goes directly)
- **Selective mode**: Only domains in the list use the VPN connection

#### Managing Domains

- **Filter/Add**: Use the text field at the top to filter existing domains or enter a new domain name
- **Append**: Click the "Append" button to add the domain from the text field to the exclusion list
- **Remove**: Click the "X" button next to any domain to remove it from the list

### Import/Export

The Import and Export buttons allow you to save and restore domain exclusion lists.

#### ExportClick "Export" to save the current domain list to a file:

1. A dialog appears showing existing export files (if any)
2. Click on an existing file to select it, or type a new filename
3. Choose an action:
   - **Append**: Add the current domains to the selected file (creates new file if it doesn't exist)
   - **Overwrite**: Replace the file contents with the current domains (asks for confirmation if file exists)
   - **Cancel**: Close the dialog without saving

**Note**: Only the currently filtered/visible domains are exported. If you have a filter active, only matching domains will be saved. Clear the filter to export all domains.

Export files are stored in: `~/.local/share/adgui/site-exclusions/`

#### Import

Click "Import" to load domains from a previously exported file:

1. A dialog appears listing all available export files
2. Click on a file to import its contents
3. Domains from the file are added to the current exclusion list
4. Duplicate domains (already in the list) are automatically skipped

The import operation shows a progress indicator and refreshes the list upon completion.

#### File Format

Export files are plain text with one domain per line. You can also create or edit these files manually:

```
example.com
subdomain.example.org
another-site.net
```