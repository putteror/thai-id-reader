# Thai ID Card Reader Microservice

A robust Go-based microservice for reading Thai National ID cards using a smart card reader (PCSC). This project implements a 3-layer architecture (Adapter, Service, Handler) and provides both a Command Line Interface (CLI) and an HTTP JSON API.

## Features

- **Data Extraction**: Reads Citizen ID, Name (TH/EN), Birth Date, and Gender.
- **Photo Extraction**: Retrieves the cardholder's photo as a Base64-encoded JPEG.
- **Hardware Resilience**: Automatic `SCARD_RESET_CARD` logic to handle unresponsive card sessions or quick card swaps.
- **HTTP JSON API**: A Gin-powered backend that handles connections on-demand per request.
- **Cross-Platform UI**: Embedded HTML dashboard for instant verification via browser.

## Prerequisites

- **Go**: Version 1.22 or higher.
- **PCSC Daemon**: 
  - **macOS**: Built-in (pcscd).
  - **Linux**: Install `pcscd` (e.g., `sudo apt install pcscd libpcsclite-dev`).
  - **Windows**: Smart Card service must be running.
- **Smart Card Reader**: A standard USB smart card reader compatible with Thai ID cards.

## Setup

1. **Clone the repository**:
   ```bash
   git clone github.com/putteror/thai-id-reader
   cd thai-id-reader
   ```

2. **Install dependencies**:
   ```bash
   go mod tidy
   ```

3. **Configure Environment**:
   Create a `.env` file in the root directory:
   ```env
   PORT=8080
   ```

## Usage

### 1. HTTP JSON API & Web UI
Start the web service:
```bash
go run cmd/http/main.go
```
- **Web UI**: Access `http://localhost:8080` in your browser.
- **JSON API**: Call `GET http://localhost:8080/api/read` to get raw JSON data.

### 2. Command Line Interface (CLI)
Run the one-time scan utility:
```bash
go run cmd/main.go
```

## API Documentation

### Read Card Data
- **Endpoint**: `/api/read`
- **Method**: `GET`
- **Response**:
```json
{
  "error": false,
  "data": {
    "CitizenID": "1100000000000",
    "FullNameTH": "นาย สมชาย เข็มกลัด",
    "FullNameEN": "Mr. Somchai Khemklad",
    "BirthDate": "25300101",
    "Gender": "1",
    "Photo": "/9j/4AAQSkZJRg..."
  }
}
```

## Project Structure

- `cmd/`: Entry points for CLI and HTTP server.
- `internal/adapter/pcsc/`: Low-level PCSC communication and hardware reset logic.
- `internal/service/`: Business logic, APDU commands, and TIS-620/UTF-8 decoding.
- `internal/handler/`: 
  - `cli/`: Console output formatting.
  - `http/`: Gin routes and embedded HTML UI.

## License
MIT
