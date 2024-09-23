# wav-track-extract

**wav-track-extract** is a command-line tool designed to process directories of multi-channel (interweaved) WAV files. It extracts each individual track from across multiple files and combines them into separate, single-track WAV files for easier management and further processing.

This tool is useful for extracting X32 X-LIVE SD card recordings, which are saved as multi-channel WAV files, into individual tracks for mixing, editing, and mastering.

## Features

- Supports Windows, Linux, and macOS (Intel and Apple Silicon)
- Processes folder of multi-channel (interweaved) WAV files
- Outputs separate WAV files for each track

## Usage

After downloading the binary for your platform, you can run the tool using the following command:

### Mac / Linux:

_Run in **Terminal**:_
```bash
wav-track-extract --in <input-directory> --out <output-directory>
```

### Windows:

_Run in **Command Prompt**:_
```bash
wav-track-extract.exe --in <input-directory> --out <output-directory>
```

- `--in <folder>`: Directory containing the input WAV files. (Defaults to the current directory if not provided.)
- `--out <folder>`: Directory where the output WAV files will be saved. (Required)
- `--force`: Overwrite existing output files.

## Installation

You can download pre-built binaries for your operating system from the releases section. Use the following commands to download and set up the tool for your platform:

### Windows:
1. Download the `wav-track-extract.exe` file: https://raw.githubusercontent.com/calebmcelroy/wav-track-extract/master/bin/windows/amd64/wav-track-extract.exe
2. Move the `.exe` file to a directory such as `C:\Users\<Your User>`.
3. Add the directory to your `PATH` environment variable ([see screenshots](https://medium.com/@kevinmarkvi/how-to-add-executables-to-your-path-in-windows-5ffa4ce61a53)):
   - Right-click on the Windows Logo and click "System".
   - Click on "Advanced System Settings".
   - Click the "Environment Variables" button.
   - In the "System variables" section, scroll down and select "Path", then click "Edit".
   - Click "New" and add `C:\Users\<Your User>` (or wherever you've placed `wav-track-extract.exe`).
   - Click **OK** to save.

### macOS (Apple Silicon):
Run in Terminal:
```bash
sudo curl -L -o /usr/local/bin/wav-track-extract https://raw.githubusercontent.com/calebmcelroy/wav-track-extract/master/bin/darwin/arm64/wav-track-extract && sudo chmod +x /usr/local/bin/wav-track-extract
```

### macOS (Intel):
Run in Terminal:
```bash
sudo curl -L -o /usr/local/bin/wav-track-extract https://raw.githubusercontent.com/calebmcelroy/wav-track-extract/master/bin/darwin/amd64/wav-track-extract && sudo chmod +x /usr/local/bin/wav-track-extract
```

### Linux:
Run in Terminal:
```bash
sudo curl -L -o /usr/local/bin/wav-track-extract https://raw.githubusercontent.com/calebmcelroy/wav-track-extract/master/bin/linux/amd64/wav-track-extract && sudo chmod +x /usr/local/bin/wav-track-extract
```

## Building From Source

If you prefer to build the project from source, you can use the provided `Makefile` to compile the binaries for all platforms.

1. Clone the repository:
   `git clone https://github.com/calebmcelroy/wav-track-extract.git`

2. Install the Go programming language:
   https://golang.org/doc/install

3. Navigate to the project directory:
   `cd wav-track-extract`

4. Build for all platforms:
   `make`

The compiled binaries will be located in the `bin/` folder for each platform (e.g., `bin/windows/amd64`, `bin/linux/amd64`, `bin/darwin/amd64`, `bin/darwin/arm64`).

## License

This project is licensed under the MIT License.