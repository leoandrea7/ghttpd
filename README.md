# ghttpd (Go HTTP Daemon)

An ultra-lightweight and minimalist HTTP server written in Go, designed to serve static files and generate directory listings. This project demonstrates low-level socket usage and concurrency management through a worker pool, without relying on the `net/http` package.  

## Features  

- **Ultra-Lightweight:** Minimal implementation using raw TCP sockets and manual HTTP request parsing.  
- **Static File Serving:** Serves static files and generates HTML-based directory listings.  
- **Worker Pool:** Concurrency managed through a configurable number of worker goroutines to prevent uncontrolled spawning.  
- **Configurable:** Set the port, directory to serve, and number of workers via command-line flags.  

## Running

### Container

Build image
```sh
podman build -t ghttpd:0.0.1 .
```

Run container
```sh
podman run -p 8080:8080 -v ./public:/data:z ghttpd:0.0.1 -d=/data
```

### No Container
You need to have [Go](https://go.dev/dl/) installed.
Run ghttpd server using the following command:
```sh
go run main.go -p 8080 -d /path/to/directory -w 4
```

Or build the binary and execute it:
```sh
go build -o ghttpd main.go
./ghttpd -p 8080 -d /path/to/directory -w 4
```

Once running, access the server in your browser:

```sh
http://localhost:8080
```

If a directory is requested, ghttpd generates an HTML-based directory listing. If a file is requested, it serves the file with the appropriate Content-Type based on its extension


## Command-Line Flags

| Flag  | Description | Default |
|-------|------------|---------|
| `-p`  | Port to listen on | `8080` |
| `-d`  | Directory to serve | `.` (current directory) |
| `-w`  | Number of worker goroutines | Number of CPU cores |

## Example Usage
Serve the current directory on port 8000, with 4 workers and specific directory:

```sh
./ghttpd -p 8000 -d /var/wwww -w 4
```