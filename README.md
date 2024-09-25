# wav-extract

**wav-extract** is a command-line tool designed to process multi-channel (interweaved) WAV files. It extracts each channel from across multiple files and combines them into separate, single-track (mono or stereo) WAV files for easier management and further processing.

Use case: this tool is useful for extracting X32 X-LIVE SD card recordings into individual tracks for mixing, editing, and mastering.

## Features

- Supports Windows, Linux, and macOS (Intel and Apple Silicon)
- Processes multi-channel interleaved WAV files from a file or folder.
- Outputs separate WAV files for each track
- Support extracting mono & stereo tracks

## Usage

After downloading the binary for your platform, you can run the tool using the following command:

### Mac / Linux:

_Run in **Terminal**:_
```bash
wav-extract --in <folder|file> --out <folder>
```

### Windows:

_Run in **Command Prompt**:_
```bash
wav-extract.exe --in <folder|file> --out <folder>
```

- `--in <folder|file>`: Folder or file containing the input WAV files. (Defaults to the current folder if not provided.)
- `--out <folder>`: Folder where the output WAV files will be saved. (Required)
- `--stereo "1/2,5/6"`: Specify stereo pairs using comma-separated channel numbers (e.g., “1/2,5/6”). Channels not included in these pairs will be extracted as mono. By default, all channels are extracted as mono if no stereo pairs are specified. This cannot be used in conjunction with --channel.
- `--channels "1/2,5"`: Specify stereo pairs & mono channels to be extracted using comma-separated channel numbers (e.g., "1/2,5"). Channels not included will NOT be extracted. This cannot be used in conjunction with --stereo.
- `--force`: Overwrite existing output files.

## Installation

You can download pre-built binaries for your operating system from the releases section. Use the following commands to download and set up the tool for your platform:

### Windows:
1. Download the `wav-extract.exe` file: https://raw.githubusercontent.com/calebmcelroy/wav-extract/master/bin/windows/amd64/wav-extract.exe
2. Move the `.exe` file to a folder such as `C:\Users\<Your User>`.
3. Add the folder to your `PATH` environment variable ([see screenshots](https://medium.com/@kevinmarkvi/how-to-add-executables-to-your-path-in-windows-5ffa4ce61a53)):
   - Right-click on the Windows Logo and click "System".
   - Click on "Advanced System Settings".
   - Click the "Environment Variables" button.
   - In the "System variables" section, scroll down and select "Path", then click "Edit".
   - Click "New" and add `C:\Users\<Your User>` (or wherever you've placed `wav-extract.exe`).
   - Click **OK** to save.

### macOS (Apple Silicon):
Run in Terminal:
```bash
sudo curl -L -o /usr/local/bin/wav-extract https://raw.githubusercontent.com/calebmcelroy/wav-extract/master/bin/darwin/arm64/wav-extract && sudo chmod +x /usr/local/bin/wav-extract
```

### macOS (Intel):
Run in Terminal:
```bash
sudo curl -L -o /usr/local/bin/wav-extract https://raw.githubusercontent.com/calebmcelroy/wav-extract/master/bin/darwin/amd64/wav-extract && sudo chmod +x /usr/local/bin/wav-extract
```

### Linux:
Run in Terminal:
```bash
sudo curl -L -o /usr/local/bin/wav-extract https://raw.githubusercontent.com/calebmcelroy/wav-extract/master/bin/linux/amd64/wav-extract && sudo chmod +x /usr/local/bin/wav-extract
```

## Building From Source

If you prefer to build the project from source, you can use the provided `Makefile` to compile the binaries for all platforms.

1. Clone the repository:
   `git clone https://github.com/calebmcelroy/wav-extract.git`

2. Install the Go programming language:
   https://golang.org/doc/install

3. Navigate to the project folder:
   `cd wav-extract`

4. Build for all platforms:
   `make`

The compiled binaries will be located in the `bin/` folder for each platform (e.g., `bin/windows/amd64`, `bin/linux/amd64`, `bin/darwin/amd64`, `bin/darwin/arm64`).

## License

This project is licensed under the MIT License.