
# OBB data sender

**A graphical utility built with Fyne to simplify pushing files (like **`.obb` or other data files) to specific application folders on an Android device using ADB.

## Download

| **Operating System** | **Version** | **Download Link**                                                                                |
| -------------------------- | ----------------- | ------------------------------------------------------------------------------------------------------ |
| **Windows (x64)**    | **v1.0.0**  | [Download](https://github.com/Blackmanx/obb_data_sender/releases/download/v1.0.1/obb_data_sender.exe "null") |

## Features

* **List Device Packages** **: Automatically lists all third-party applications installed on the connected device.**
* **Package Search** **: A real-time search bar to quickly filter and find the desired application.**
* **Destination Choice** **: Easily select whether to send the file to the **`data/files` or `/obb` directory of the application's data folder.
* **Custom File Picker** **: A large, user-friendly file browser with its own search functionality to easily locate files on your computer.**
* **Progress Bar** **: Visual feedback on the file transfer progress.**
* **Multi-language** **: UI available in English and Spanish.**

## Usage

1. **Connect your device** **: Ensure your Android device is connected to your computer via USB with ** **USB Debugging enabled** **.**
2. **Install ADB** **: Make sure you have **[Android Debug Bridge (ADB)](https://www.google.com/search?q=https://developer.android.com/studio/command-line/adb "null") installed and that its location is included in your system's PATH environment variable.
3. **Run the application** **: Execute the **`obb_data_sender.exe` file (on Windows) or the corresponding binary for your OS.
4. **List Packages** **: Click the "List Packages" button to populate the list with apps from your device.**
5. **Select an App** **: Click on an application in the list to select it as the destination.**
6. **Choose Destination** **: Select the target subfolder (**`data/files` or `/obb`).
7. **Send File** **: Click the "Send File..." button, browse for the file you want to transfer using the custom file picker, and click "Open".**
8. **The transfer will begin, and the progress bar will show its status.**

## Building from Source

**To build the application yourself, you will need:**

* **Go (version 1.18 or later)**
* **A C compiler (like GCC/MinGW)**
* **Fyne dependencies**

**To build for your current OS:**

```
go build .

```

**To cross-compile for Windows from Linux/WSL:**

**First, install the Fyne CLI tool and the MinGW cross-compiler:**

```
go install fyne.io/tools/cmd/fyne@latest
sudo apt-get install mingw-w64

```

**Then, package the application:**

```
fyne package -os windows -name "obb_data_sender"

```

**This will create a **`fyne-windows-amd64` directory containing the `.exe` file and its icon.
