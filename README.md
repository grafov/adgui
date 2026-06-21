# adgui

Simple GUI for control CLI for AdGuard VPN. Linux XLibre/Wayland.

**Works partially yet. Development just in progress.**

## Localization

The interface language is selected automatically from the system locale (Fyne i18n). Supported UI languages: English (`en`), Russian (`ru`), and Esperanto (`eo`). English is used as fallback when no matching translation is available.

## Installation

It tries to install into protected directory `/usr/local/bin` that requires root privileges.
Use "sudo", "doas" or other appropriate command with SUDO environvent dir:

`SUDO=sudo make install`

Or use `PREFIX` for installing to another directory, for example under home:

`PREFIX=~/bin make install`

## Features

### Site Exclusions (Domains Tab)

The Domains tab in the dashboard allows you to manage site exclusions for AdGuard VPN. You can configure which domains bypass or use the VPN connection.

#### Exclusion Modes
- **General mode**: Domains in the list are excluded from VPN (traffic goes directly)
- **Selective mode**: Only domains in the list use the VPN connection

#### Automatic Persistence
The exclusion lists are separated by mode (General and Selective) and automatically persist to the following local files on any change (Add, Paste, Import, Remove, Clear):
- General mode: `~/.config/adgui/site-exclusions/general.txt`
- Selective mode: `~/.config/adgui/site-exclusions/selective.txt`

When you switch exclusion modes, the current active list is saved to its corresponding file, and the list for the new mode is automatically loaded and applied to the CLI.

#### Managing Domains

- **Filter/Add**: Use the text field at the top to filter existing domains or enter a new domain name
- **Append**: Click the "Append" button to add the domain from the text field to the exclusion list
- **Remove**: Click the "X" button next to any domain to remove it from the list

### Import/Export

The Import and Export buttons allow you to save and restore domain exclusion lists for the **current exclusion mode** (General or Selective).

#### Export

Click **Export** to save domains from the current mode to a file:

1. A system save dialog opens with the default filename `<mode>.adgui` (`general.adgui` or `selective.adgui`)
2. Choose the destination path and filename (default extension: `.adgui`)
3. The file is saved with the exported domains

**Note**: Only the currently filtered/visible domains are exported. If you have a filter active, only matching domains will be saved. Clear the filter to export all domains in the current mode.

#### Import

Click **Import** to load domains from a file into the **current exclusion mode**:

1. A system open dialog opens (any file extension)
2. Select a file with one domain per line
3. New domains are added to the current mode list in adgui and immediately applied to AdGuard VPN
4. Duplicate domains (already in the list) are automatically skipped

The import operation shows a progress indicator and refreshes the list upon completion. Imported domains are persisted to the current mode file (`general.txt` or `selective.txt`).

### Migration

If you have old unified plain-text exclusion files, you can migrate them to the new mode-specific files using the provided Python script:

```bash
python3 scripts/migrate-site-exclusions.py --target-mode [general|selective]
```

By default, the script scans the legacy `~/.local/share/adgui/site-exclusions/` directory and merges all files (excluding `general.txt` and `selective.txt`) into the target mode file at `~/.config/adgui/site-exclusions/` with automatic deduplication. You can also specify input files explicitly using the `--input <path>` flag.

#### File Format

Export/import files are plain text with one domain per line. The default export extension is `.adgui`. You can also create or edit these files manually:

```
example.com
subdomain.example.org
another-site.net
```

### Location Bookmarks (Connect To)

The **Connect To...** location selector lets you bookmark VPN locations with the star column on the right. Bookmarked locations are saved to:

- `~/.config/adgui/bookmarks`

Click the **★** column header to toggle sorting bookmarked locations first. Click a row star to add or remove a bookmark without connecting.

Country flags in the location list use SVG assets from [lipis/flag-icons](https://github.com/lipis/flag-icons) (MIT license), embedded in the application binary.