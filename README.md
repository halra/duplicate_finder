# duplicate_finder

This command-line tool is designed to help you find and manage duplicate files in a specified folder efficiently. It uses the MD5 hash of files to identify duplicates, and it provides options to list, move, or delete duplicate files based on your preferences.

## Features

- Fast and efficient duplicate file detection using concurrent processing.
- User-friendly command-line interface for interactive file management.
- Supports moving and deleting duplicate files.
- Works on Windows, macOS, and Linux.

## Getting Started

### Prerequisites

- [Go](https://golang.org/) (1.16 or higher)

### Usage

1. Open your terminal/command prompt and navigate to the source folder where the project is located.

#### For Windows:

2. Build the executable using Go:

   ```
   go build
   ```

3. After the build is successful, you can run the program by executing the generated executable:

   ```
   duplicate_finder.exe
   ```

#### For Linux and macOS:

2. Build the executable using Go:

   ```
   go build
   ```

3. After the build is successful, you can run the program by executing the generated executable:

   ```
   ./duplicate_finder
   ```

4. Follow the on-screen prompts to manage the duplicate files. You can list, move, delete, or ignore duplicates based on your preferences.


## License

This project is licensed under the MIT License - see the [LICENSE](LICENSE) file for details.

## Acknowledgments

- This tool is built with Go and utilizes concurrent processing for efficient file scanning.
- Special thanks to the Go community for their valuable contributions to open-source software.

