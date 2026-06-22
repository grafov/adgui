# AdGUI for the Adguard VPN

**Languages:** [English](README.md) · [Русский](README.ru.md) · [Esperanto](README.eo.md)

Simple GUI to control the CLI for AdGuard VPN on Linux desktop (XLibre/X11 or Wayland).
AdGuard officially offers GUIs for Mac, Android, and Windows, but Linux is missing one :(

> The project doesn't offer VPN functionality. This is just a helper UI that wraps
> the real VPN application (`adguardvpn-cli`) for more comfortable use in a
> desktop environment. The project has no relation to Adguard or any of their
> products.

The GUI closely resembles the features offered by AdGuard VPN on Linux. Sadly,
the Linux version of AdGuard VPN has fewer features than its counterparts on
Mac/Windows.

Work in progress but the application is fully functional.

![connection info](doc/scr2.png)

![region check](doc/scr1.png)

## Localization

The interface language is selected automatically from the system locale (Fyne i18n). Supported UI languages: English (`en`), Russian (`ru`), and Esperanto (`eo`). English is used as fallback when no matching translation is available.
Other languages could be added later.

## Installation

No binaries provided at the moment. You should need to have Golang development environment on your machine.

It tries to install into protected directory `/usr/local/bin` that requires root privileges. Use "sudo", "doas" or other appropriate command with SUDO environvent dir:

`SUDO=sudo make install`

Or use `PREFIX` for installing to another directory, for example under home:

`PREFIX=~/bin make install`

## Features

Fistly login to your Adguard account with `adguardvpn-cli`. I missed this part in GUI for simplicity, because you need it only once.

### Tray support

The set of actions available in application tray icon: show dashboard, connect to a location, connect to previous location, site-exclusions configuration, disconnect VPN.

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

### IP Region (Country Detection)

This feature implemented just in Adgui and not a part of Adguard VPN. The feature could help you analyze effeciency of your VPN connection. The **IP Region** tab on the dashboard checks how GeoIP databases and popular web services classify your current egress IP address. Use it to verify that AdGuard VPN routes traffic through the expected country, or to see whether different services disagree about your location.

The check runs only when you press **Where am I?** — opening the tab does not contact the network. While a scan is in progress, a progress bar shows the current service and attempt counter; press the button again to cancel.

#### Results

- **Summary** — top countries by how many services reported each ISO code (IPv4 and IPv6 percentages when both are available)
- **Services table** — per-service country codes for IPv4 and IPv6
- **VPN comparison** — when connected, compares the selected VPN location with the consensus from external checks; mismatches are marked with `!` in the table

Primary probes query GeoIP APIs (MaxMind, ipinfo.io, Cloudflare, ip-api.com, and others). Custom probes infer region from responses of popular sites (Google, YouTube, Netflix, Spotify, Steam, and others). Logic is ported from [Davoyan/ipregion](https://github.com/Davoyan/ipregion) (MIT license).

#### Optional API Keys

Some services for checking of region accept your own API keys. Create `~/.config/adgui/service-keys` (INI format) with any of:

```ini
IPREGISTRY_KEY=your_key
GEOAPIFY_KEY=your_key
SPOTIFY_CLIENT_ID=your_client_id
SPOTIFY_API_KEY=your_api_key
AIRPORT_CODES_AUTH=your_token
```

If the file or a key is missing, built-in demo defaults are used where available; other services work without keys.

## AI code

I actively use LLMs for generating large parts of code, tests, code review, localization and documentation for this project. It was an experiment to create a GUI (using the Fyne framework) in Go with LLMs. Mostly successful, though I needed to fix some parts manually. All the code reviewed by me.

## Starware

This software is a starware :) If you find the code useful, don’t forget to vote for this repo with a star! ⭐

## License

Under terms of GPL v3.
