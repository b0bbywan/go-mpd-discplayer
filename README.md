# go-mpd-discplayer

[![Go Reference](https://pkg.go.dev/badge/github.com/b0bbywan/go-mpd-discplayer.svg)](https://pkg.go.dev/github.com/b0bbywan/go-mpd-discplayer)

`mpd-discplayer` is a Go-based client designed for seamless disc playback with an MPD (Music Player Daemon) server. It automatically handles disconnections and uses [go-disc-cuer](https://github.com/b0bbywan/go-disc-cuer/) to generate CUE files for inserted audio discs.


## Features

- **Audio Disc Playback Automation**: Monitors for inserted audio discs and generates CUE files using go-disc-cuer before playing on mpd server
- **USB Stick Playback Automation**: Monitors removable usb media to play them on mpd.server
- **Reconnection Logic**: Automatically reconnects to the MPD server if the connection is lost.
- **Reliable Connection Management**: Thread-safe operations to manage MPD client connections and operations.
- **Configurable Settings**: Supports configuration via a unified config file or environment variables.
- **Audio Notifications**: Audio notifications are triggered on device insertion and removal, and critical errors encountered while processing those events


## Installation

To use `mpd-discplayer`, you'll need to have Go installed on your system. If you don’t have Go installed, follow the instructions at [https://golang.org/doc/install](https://golang.org/doc/install).

### Clone the repository

```bash
git clone https://github.com/b0bbywan/go-mpd-discplayer.git
cd go-mpd-discplayer
```

### Install dependencies
Ensure libgudev and libdiscid are installed. Run the following commands based on your OS:

```bash
# Debian
sudo apt install libdiscid0 libdiscid-dev libgudev-1.0-0 libgudev-1.0-dev libasound2-dev
# Fedora
sudo dnf install libdiscid libdiscid-devel libgudev libgudev-devel alsa-lib-devel
```

### Build the Project
```bash
go build -o mpd-discplayer
```


## Usage

Simply run the program. mpd-discplayer will:

- Monitor the system for an audio disc or USB stick insertion.
- Automatically use go-disc-cuer to generate a CUE file for inserted audio disc.
- Play the disc or USB on the MPD server.
- Trigger audio notifications on device insertion and removal, and errors.

```bash
./mpd-discplayer
```


## Configuration

`mpd-discplayer` can be configured using a YAML configuration file or environment variables. This allows flexibility in managing settings for both the MPD server connection and the `disc-cuer` tool. Below is a detailed explanation of the configuration options and how to use them.
Visit [go-disc-cuer](https://github.com/b0bbywan/go-disc-cuer/) for more informations

### Configuration File

The configuration file is expected in one of the following locations:
1. `/etc/mpd-discplayer/config.yml` (system-wide configuration)
2. `~/.config/mpd-discplayer/config.yml` (user-specific configuration)

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

```

#### MPD Connection Options
Under the MPDConnection key, you configure how mpd-discplayer connects to the MPD server:

- **Type**:
The type of connection to use. Supported values:
	- `"unix"`: For a Unix socket connection.
	- `"tcp"`: For a TCP connection over the network. (default)
- **Address**:
	- For Type: `"unix"`, this is the path to the MPD socket file (e.g., `/var/run/mpd/socket`).
	- For Type: `"tcp"`, this is the <hostname>:<port> of the MPD server (e.g., `127.0.0.1:6600`) *(default)*.


#### Notifications Options
- **AudioBackend**: `"pulse"` *(default)*, `"alsa"` or `"none` (disable notifications).
- **PulseServer**: Check [Pulseaudio Server String doc](https://www.freedesktop.org/wiki/Software/PulseAudio/Documentation/User/ServerStrings/)
- **SoundsLocation**: `"/usr/local/share/mpd-discplayer"` *(default)*. No default sounds are provided at the moment. Notifications expect `in.mp3`, `out.mp3` and `error.mp3` to be present in the specified folder or notifications will be disabled.


### Environment Variables

If a configuration file is not provided, you can use environment variables to set the same options. Below is the list of supported variables and their defaults (if applicable):

| Environment Variable           | YAML equivalent                          | Default Value                  |
|--------------------------------|--------------------------------------|--------------------------------|
| `MPD_DISCPLAYER_GNUHELLOEMAIL`     | `gnuHelloEmail`      | *(no default,  empty value disable the integration)*      |
| `MPD_DISCPLAYER_GNUDBURL`          | `gnuDbUrl`           | `https://gnudb.gnudb.org`    |
| `MPD_DISCPLAYER_MPDCONNECTION_TYPE`         | `MPDConnection.Type` | `tcp`                        |
| `MPD_DISCPLAYER_MPDCONNECTION_ADDRESS`      | `MPDConnection.Address` | `127.0.0.1:6600`   |
| `MPD_DISCPLAYER_MPDCONNECTION_RECONNECTWAIT`      | `MPDConnection.ReconnectWait` | `30` (in seconds)          |
| `MPD_DISCPLAYER_MPDLIBRARYFOLDER` | `MPDLibraryFolder` | `/var/lib/mpd/music`
| `MPD_DISCPLAYER_DISCSPEED` | `DiscSpeed` | `12`
| `MPD_DISCPLAYER_SOUNDSLOCATION` | `SoundsLocation` | `/usr/local/share/mpd-discplayer`
| `MPD_DISCPLAYER_AUDIOBACKEND` | `AudioBackend` | `pulse`
| `MPD_DISCPLAYER_PULSESERVER` | `PulseServer` | *(Default to "", e.g. local unix socket)*

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
