# go-mpd-discplayer

[![Go Reference](https://pkg.go.dev/badge/github.com/b0bbywan/go-mpd-discplayer.svg)](https://pkg.go.dev/github.com/b0bbywan/go-mpd-discplayer)

`mpd-discplayer` is a Go-based client designed for seamless integration with MPD (Music Player Daemon). It automates disc and USB playback by generating CUE files for audio discs and managing removable media. It also handles disconnections gracefully and provides configurable audio notifications.

## Features

- **Automated Audio Disc Playback:** Detects and plays inserted audio discs using `go-disc-cuer` for CUE file generation.
- **USB Media Playback:** Monitors removable USB drives and plays media files on the MPD server.
- **Robust Reconnection Logic:** Automatically reconnects to the MPD server if the connection is lost.
- **Flexible Configuration:** Supports configuration via YAML files or environment variables.
- **Audio Notifications:** Plays sound notifications for device events (insertion, removal) and critical errors.

## Installation

### From the apt repository (Debian/Raspberry Pi OS)

```bash
curl -fsSL https://apt.odio.love/key.gpg | sudo gpg --dearmor -o /usr/share/keyrings/odio.gpg
echo "deb [signed-by=/usr/share/keyrings/odio.gpg] https://apt.odio.love stable main" | sudo tee /etc/apt/sources.list.d/odio.list
sudo apt update
sudo apt install mpd-discplayer
```

### From a release binary

Download the binary for your architecture from the [releases page](https://github.com/b0bbywan/go-mpd-discplayer/releases) and install the required runtime libraries:

```bash
# Debian Bookworm
sudo apt install libcdparanoia0 libdiscid0 libgudev-1.0-0

# Debian Trixie / Raspberry Pi OS (latest)
sudo apt install libcdio-paranoia2t64 libdiscid0 libgudev-1.0-0
```

### Build from source

#### Native build (on the target machine)

Install the dev libraries first:

```bash
# Debian
sudo apt install gcc libdiscid-dev libgudev-1.0-dev libasound2-dev pkg-config

# Fedora
sudo dnf install gcc libdiscid-devel libgudev-devel alsa-lib-devel pkgconf
```

Then:

```bash
git clone https://github.com/b0bbywan/go-mpd-discplayer.git
cd go-mpd-discplayer

go build -o mpd-discplayer .
```

#### Cross-compilation (Docker multi-arch)

Requires Docker with buildx and QEMU support. `.deb` packaging requires [nfpm](https://nfpm.goreleaser.com/).

```bash
git clone https://github.com/b0bbywan/go-mpd-discplayer.git
cd go-mpd-discplayer
make setup

# Build for all architectures (amd64, arm64, armv6, armv7)
make build-all

# Or for a specific architecture
make build-arm64
make build-armv6   # Raspberry Pi 1 / Zero
make build-armv7   # Raspberry Pi 2 / 3 (32-bit)

# Build .deb packages (requires nfpm, binaries must exist in dist/)
make deb-all       # build-all + deb for all architectures
make deb-armv6     # deb only, binary must already be in dist/
```

Binaries are output to `dist/`.

## Usage

Once installed, `mpd-discplayer` automatically detects devices and interacts with the MPD server. Run the following command to start:

```bash
./mpd-discplayer
```

### Using Systemd (Optional)
To run mpd-discplayer as a systemd service:

```bash
sudo cp share/mpd-discplayer.service /usr/lib/systemd/user/
systemctl --user daemon-reload
systemctl --user enable mpd-discplayer # enable on user login
systemctl --user start mpd-discplayer
```

## MPD Configuration
- To enable MPD-Discplayer some configuration is necessary :
	- It's recommended to run mpd and mpd-discplayer with the same user:
		- MPD-Discplayer needs to be able to write in MPD music_directory (at least in `MPDCueSubfolder`  for cue file storage and `MPDUSBSubfolder` for `symlink` mounting.
		- MPD also needs to be able to read files and directories created by MPD-Discplayer
	- **Disc Plyback**:
		- `input { plugin "cdio_paranoia" }`: allow MPD to play audio disc
		- `playlist_plugin { name "cue" enabled "true" as_folder "true" }`: allow audio disc covers display in MPD clients
	- **USB Playback**: Those settings are only necessary if using `mpd` mounting, `symlink` mounting doesn't need it.
		- `neighbors { plugin "udisks" }`: Enable udisk mounting MPD feature
		- `database { plugin "simple" path "~/.local/share/mpd/db" cache_directory "~/.local/share/mpd/cache" }`: Mandatory with neighbors plugins. Old `db_file` setting won't work with neighbors plugins
		- **Headless / Polkit rule**: In a headless setup, udisks2 may deny mount requests from the MPD user. Add the following rule to `/etc/polkit-1/rules.d/80-mpd-udisks.rules` (replace `<target_user>` with the user running mpd-discplayer):

```javascript
polkit.addRule(function(action, subject) {
  if ((action.id == "org.freedesktop.udisks2.filesystem-mount" ||
       action.id == "org.freedesktop.udisks2.filesystem-mount-other-seat") &&
      subject.user == "<target_user>") {
      return polkit.Result.YES;
  }
});
```

```
# Database #######################################################################
#
database {
	plugin "simple"
	path "~/.local/share/mpd/db"
	cache_directory "~/.local/share/mpd/cache"
}

# Input #######################################################################
#
input {
	plugin	"cdio_paranoia"
}
#
###############################################################################

# Neighbors ###################################################################
#
neighbors {
	plugin "udisks"
}
#
# #############################################################################

# Playlist Plugin #############################################################
#
playlist_plugin {
	name "cue"
	enabled "true"
	as_folder "true"
}
#
###############################################################################
```

## Configuration

`mpd-discplayer` can be configured using a YAML configuration file or environment variables. This allows flexibility in managing settings for both the MPD server connection and the `disc-cuer` tool. Below is a detailed explanation of the configuration options and how to use them.
Visit [go-disc-cuer](https://github.com/b0bbywan/go-disc-cuer/) for more informations

### Configuration File

The configuration file is expected in one of the following locations:
1. `/etc/mpd-discplayer/config.yaml` (system-wide configuration)
2. `~/.config/mpd-discplayer/config.yaml` (user-specific configuration)

The file should be written in YAML format, and a sample structure is shown below:

```yaml
gnuHelloEmail: "your-email@example.com"
gnuDbUrl: "https://gnudb.gnudb.org"

MPDConnection:
  Type: "tcp"
  Address: "127.0.0.1:6600"
  ReconnectWait: 30
MPDLibraryFolder: "/var/lib/mpd/music"
DiscSpeed: 12
SoundsLocation: "/usr/local/share/mpd-discplayer"
AudioBackend: "pulse"
PulseServer: ""
MountConfig: "mpd"
MPDCueSubfolder: ".disc-cuer"
MPDUSBSubfolder: ".udisks"
Schedule: {}

```

#### MPD Connection Options
Under the MPDConnection key, you configure how mpd-discplayer connects to the MPD server:

- **Type**:
The type of connection to use. Supported values:
	- `"unix"`: For a Unix socket connection *(recommended as it enables MPD music_directory self discovery to set the directory for cue file and USB symlink mounting)*
	- `"tcp"`: For a TCP connection over the network. (default)

- **Address**:
	- For Type: `"unix"`, this is the path to the MPD socket file (e.g., `/var/run/mpd/socket` ) *(recommended)*.
	- For Type: `"tcp"`, this is the <hostname>:<port> of the MPD server (e.g., `127.0.0.1:6600`) *(default)*.

#### Mouting Options
For USB stick support, the content of the stick must be made available in MPD database. MPD-Discplayer supports the native mpd mouting feature, or symlinks for MPD servers that do not support this feature.
- **MountConfig**:
	- `mpd`
	- `symlink`
- **MPDLibraryFolder**: path to MPD music_directory *(self discovered when using MPD unix socket)*
- **MPDUSBSubfolder**: path inside `MPDLibraryFolder` to store symlinks to usb original mountpoints. Only with `MountConfig: "symlink"`, not used with `MountConfig: "mpd"`

#### Schedule Option
The Schedule option allows you to automate playback of MPD-compatible URIs based on a cron schedule. It currently supports the following:
	- **Webradio** URIs (e.g., HTTP streams).
	- **Audio CDs** (using cdda:// protocol).
	- **USB devices** (using mount points or symlinks).
	- **MPD Database** (use `mpc listall` to list MPD Database)

Incompatible cron will be discarded and logged on startup. A notification will be trigerred before starting the scheduled playback and if an error happens while loading the uri.

Example configuration:

```yaml
Schedule:
  # Play a reggae radio stream on weekdays at 6:30 AM
  "30 6 * * 1-5": "//hd.lagrosseradio.info/lagrosseradio-reggae-192.mp3"
  # Play an audio CD on Saturdays at 9:00 AM
  "0 9 * * 6": "cdda://"
  # Play from a USB device (symlink mount) on Sundays at 9:00 PM
  "0 21 * * 7": ".udisks/{usb_label}"
  # Play from an MPD-mounted USB device on Sundays at 9:00 PM
  "0 21 * * 7": "{usb_label}"

```
Note: Ensure that the `usb_label` matches the label of the USB device. For audio CDs, only `cdda://` protocol is supported for now.

#### Notifications Options
- **AudioBackend**: `"pulse"` *(default)*, `"alsa"` or `"none` (disable notifications).
- **PulseServer**: Check [Pulseaudio Server String doc](https://www.freedesktop.org/wiki/Software/PulseAudio/Documentation/User/ServerStrings/)
- **SoundsLocation**: `"/usr/local/share/mpd-discplayer"` *(default)*. No default sounds are provided at the moment. Notifications expect `in.pcm`, `out.pcm` and `error.pcm` to be present in the specified folder or notifications will be disabled.

### Environment Variables

If a configuration file is not provided, you can use environment variables to set the same options. Below is the list of supported variables and their defaults (if applicable):

| Environment Variable           | YAML equivalent                          | Default Value                  |
|--------------------------------|--------------------------------------|--------------------------------|
| `MPD_DISCPLAYER_GNUHELLOEMAIL`     | `gnuHelloEmail`      | *(no default,  empty value disable the integration)*      |
| `MPD_DISCPLAYER_GNUDBURL`          | `gnuDbUrl`           | `https://gnudb.gnudb.org`    |
| `MPD_DISCPLAYER_MPDCONNECTION_TYPE`         | `MPDConnection.Type` | `tcp`                        |
| `MPD_DISCPLAYER_MPDCONNECTION_ADDRESS`      | `MPDConnection.Address` | `127.0.0.1:6600`   |
| `MPD_DISCPLAYER_MPDCONNECTION_RECONNECTWAIT`      | `MPDConnection.ReconnectWait` | `30` (in seconds)          |
| `MPD_DISCPLAYER_MPDLIBRARYFOLDER` | `MPDLibraryFolder` | `/var/lib/mpd/music` |
| `MPD_DISCPLAYER_DISCSPEED` | `DiscSpeed` | `12` |
| `MPD_DISCPLAYER_SOUNDSLOCATION` | `SoundsLocation` | `/usr/local/share/mpd-discplayer` |
| `MPD_DISCPLAYER_AUDIOBACKEND` | `AudioBackend` | `pulse` |
| `MPD_DISCPLAYER_PULSESERVER` | `PulseServer` | *(Default to `""`, e.g. local pulseaudio unix socket)* | `MPD_DISCPLAYER_MOUNTCONFIG` | `MountConfig` | `mpd`
| `MPD_DISCPLAYER_MPDCUESUBFOLDER` | `MPDCueSubfolder` | `.disc-cuer` |
| `MPD_DISCPLAYER_MPDUSBSUBFOLDER` | `MPDUSBSubfolder` | `.udisks` |
| *(Unsupported)* | `Schedule` | *{}  (empty, disables scheduling)* |

#### Priority of Configuration
The configuration is loaded in the following order of priority:
- Environment variables (highest priority)
- User-specific configuration file (~/.config/mpd-discplayer/config.yml)
- System-wide configuration file (/etc/mpd-discplayer/config.yml)

## License
This project is licensed under the MIT License - see the LICENSE file for details.

## Contributing
Contributions are welcome! Feel free to fork this repository, make changes, and create a pull request.

## Acknowledgments
Thanks to [gompd](https://github.com/fhs/gompd) for the underlying MPD client implementation.
