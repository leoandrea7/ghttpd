package main

import (
	"bytes"
	"fmt"
	"net"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

// Mock net.Conn implementation for testing
type mockConn struct {
	readBuf  *bytes.Buffer
	writeBuf *bytes.Buffer
}

func newMockConn(input string) *mockConn {
	return &mockConn{
		readBuf:  bytes.NewBufferString(input),
		writeBuf: &bytes.Buffer{},
	}
}

func (m *mockConn) Read(b []byte) (n int, err error)         { return m.readBuf.Read(b) }
func (m *mockConn) Write(b []byte) (n int, err error)        { return m.writeBuf.Write(b) }
func (m *mockConn) Close() error                             { return nil }
func (m *mockConn) LocalAddr() net.Addr                      { return nil }
func (m *mockConn) RemoteAddr() net.Addr                     { return nil }
func (m *mockConn) SetDeadline(t time.Time) error            { return nil }
func (m *mockConn) SetReadDeadline(t time.Time) error        { return nil }
func (m *mockConn) SetWriteDeadline(t time.Time) error       { return nil }
func (m *mockConn) GetWrittenData() string                   { return m.writeBuf.String() }

func TestParseRequest(t *testing.T) {
	testCases := []struct {
		name          string
		input         string
		expectedMethod string
		expectedPath  string
		expectedVersion string
		shouldError   bool
	}{
		{
			name:            "Valid GET request",
			input:           "GET /index.html HTTP/1.1\r\n",
			expectedMethod:  "GET",
			expectedPath:    "/index.html",
			expectedVersion: "HTTP/1.1\r\n",
			shouldError:     false,
		},
		{
			name:          "Invalid request format - missing parts",
			input:         "GET /index.html\r\n",
			shouldError:   true,
		},
		{
			name:          "Empty request",
			input:         "",
			shouldError:   true,
		},
		{
			name:            "URL encoded path",
			input:           "GET /test%20file.html HTTP/1.1\r\n",
			expectedMethod:  "GET",
			expectedPath:    "/test file.html",
			expectedVersion: "HTTP/1.1\r\n",
			shouldError:     false,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			conn := newMockConn(tc.input)
			method, path, version, err := parseRequest(conn)

			if tc.shouldError {
				if err == nil {
					t.Errorf("Expected error but got none")
				}
			} else {
				if err != nil {
					t.Errorf("Unexpected error: %v", err)
				}
				if method != tc.expectedMethod {
					t.Errorf("Expected method %s, got %s", tc.expectedMethod, method)
				}
				if path != tc.expectedPath {
					t.Errorf("Expected path %s, got %s", tc.expectedPath, path)
				}
				if version != tc.expectedVersion {
					t.Errorf("Expected version %s, got %s", tc.expectedVersion, version)
				}
			}
		})
	}
}

func TestValidateRequest(t *testing.T) {
	testCases := []struct {
		name          string
		method        string
		version       string
		shouldError   bool
	}{
		{
			name:        "Valid GET request with HTTP version",
			method:      "GET",
			version:     "HTTP/1.1\r\n",
			shouldError: false,
		},
		{
			name:        "Invalid method",
			method:      "POST",
			version:     "HTTP/1.1\r\n",
			shouldError: true,
		},
		{
			name:        "Invalid version",
			method:      "GET",
			version:     "NOTHTTP/1.1\r\n",
			shouldError: true,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			err := validateRequest(tc.method, tc.version)

			if tc.shouldError {
				if err == nil {
					t.Errorf("Expected error but got none")
				}
			} else {
				if err != nil {
					t.Errorf("Unexpected error: %v", err)
				}
			}
		})
	}
}

func TestSendError(t *testing.T) {
	testCases := []struct {
		name        string
		statusCode  int
		message     string
		expectedResponse string
	}{
		{
			name:       "404 Not Found",
			statusCode: 404,
			message:    "Not Found",
			expectedResponse: "HTTP/1.1 404 Not Found\r\nContent-Type: text/plain\r\nContent-Length: 9\r\n\r\nNot Found",
		},
		{
			name:       "500 Internal Server Error",
			statusCode: 500,
			message:    "Internal Server Error",
			expectedResponse: "HTTP/1.1 500 Internal Server Error\r\nContent-Type: text/plain\r\nContent-Length: 21\r\n\r\nInternal Server Error",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			conn := newMockConn("")
			sendError(conn, tc.statusCode, tc.message)
			
			response := conn.GetWrittenData()
			if response != tc.expectedResponse {
				t.Errorf("Expected response:\n%s\n\nGot:\n%s", tc.expectedResponse, response)
			}
		})
	}
}

func TestSendFile(t *testing.T) {
	// Create a temporary test file
	tempContent := "This is test content."
	tempFile, err := os.CreateTemp("", "test-*.txt")
	if err != nil {
		t.Fatalf("Failed to create temp file: %v", err)
	}
	defer os.Remove(tempFile.Name())
	
	_, err = tempFile.WriteString(tempContent)
	if err != nil {
		t.Fatalf("Failed to write to temp file: %v", err)
	}
	tempFile.Close()
	
	conn := newMockConn("")
	sendFile(conn, tempFile.Name())
	
	response := conn.GetWrittenData()
	expectedHeader := fmt.Sprintf("HTTP/1.1 200 OK\r\nContent-Type: text/plain; charset=utf-8\r\nContent-Length: %d\r\n\r\n", len(tempContent))
	
	if !strings.HasPrefix(response, expectedHeader) {
		t.Errorf("Expected response to start with:\n%s\n\nGot:\n%s", expectedHeader, response)
	}
	
	if !strings.HasSuffix(response, tempContent) {
		t.Errorf("Expected response to end with content: %s", tempContent)
	}
}

func TestGenerateDirectoryListing(t *testing.T) {
	// Create a temporary directory with some files
	tempDir, err := os.MkdirTemp("", "test-dir")
	if err != nil {
		t.Fatalf("Failed to create temp directory: %v", err)
	}
	defer os.RemoveAll(tempDir)
	
	// Create a few test files in the directory
	testFiles := []string{"file1.txt", "file2.html", "subdir"}
	for _, name := range testFiles {
		path := filepath.Join(tempDir, name)
		if name == "subdir" {
			if err := os.Mkdir(path, 0755); err != nil {
				t.Fatalf("Failed to create subdirectory: %v", err)
			}
		} else {
			f, err := os.Create(path)
			if err != nil {
				t.Fatalf("Failed to create test file: %v", err)
			}
			f.Close()
		}
	}
	
	conn := newMockConn("")
	generateDirectoryListing(conn, "/testpath", tempDir)
	
	response := conn.GetWrittenData()
	
	// Check that response is an HTTP 200 OK with HTML content type
	if !strings.Contains(response, "HTTP/1.1 200 OK") {
		t.Errorf("Response doesn't contain success status code")
	}
	
	if !strings.Contains(response, "Content-Type: text/html") {
		t.Errorf("Response doesn't have HTML content type")
	}
	
	// Check that all file names are present in the HTML
	for _, fileName := range testFiles {
		if !strings.Contains(response, fileName) {
			t.Errorf("File %s not found in directory listing", fileName)
		}
	}
}

func TestHandleConnection(t *testing.T) {
	// Set up initial directory for testing
	originalDir := dir
	tempDir, err := os.MkdirTemp("", "test-server")
	if err != nil {
		t.Fatalf("Failed to create temp directory: %v", err)
	}
	defer os.RemoveAll(tempDir)
	defer func() { dir = originalDir }()
	
	dir = tempDir
	
	// Create a test file in the temp directory
	testFileName := "test.txt"
	testContent := "Test file content"
	testFilePath := filepath.Join(tempDir, testFileName)
	if err := os.WriteFile(testFilePath, []byte(testContent), 0644); err != nil {
		t.Fatalf("Failed to create test file: %v", err)
	}
	
	testCases := []struct {
		name          string
		request       string
		expectedCode  string
		expectedContentType string
		checkContent  bool
		expectedContent string
	}{
		{
			name:         "Valid file request",
			request:      fmt.Sprintf("GET /%s HTTP/1.1\r\n", testFileName),
			expectedCode: "HTTP/1.1 200 OK",
			expectedContentType: "text/plain",
			checkContent: true,
			expectedContent: testContent,
		},
		{
			name:         "Directory listing",
			request:      "GET / HTTP/1.1\r\n",
			expectedCode: "HTTP/1.1 200 OK",
			expectedContentType: "text/html",
			checkContent: true,
			expectedContent: testFileName,  // Directory listing should contain our test file
		},
		{
			name:         "File not found",
			request:      "GET /nonexistent.txt HTTP/1.1\r\n",
			expectedCode: "HTTP/1.1 404 Not Found",
			checkContent: false,
		},
		{
			name:         "Invalid method",
			request:      "POST / HTTP/1.1\r\n",
			expectedCode: "HTTP/1.1 400",
			checkContent: false,
		},
		{
			name:         "Invalid protocol",
			request:      "GET / FTP/1.1\r\n",
			expectedCode: "HTTP/1.1 400",
			checkContent: false,
		},
	}
	
	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			conn := newMockConn(tc.request)
			handleConnection(conn)
			
			response := conn.GetWrittenData()
			
			if !strings.Contains(response, tc.expectedCode) {
				t.Errorf("Expected response to contain %s, got: %s", tc.expectedCode, response)
			}
			
			if tc.expectedContentType != "" && !strings.Contains(response, "Content-Type: "+tc.expectedContentType) {
				t.Errorf("Expected Content-Type %s not found in response", tc.expectedContentType)
			}
			
			if tc.checkContent && !strings.Contains(response, tc.expectedContent) {
				t.Errorf("Expected content %s not found in response", tc.expectedContent)
			}
		})
	}
}