package api

import (
	"bytes"
	"mime/multipart"
	"testing"
)

func TestValidImageContentRejectsExtensionMismatch(t *testing.T) {
	fileHeader := multipartHeader(t, "image.png", []byte("not an image"))

	if validImageContent(fileHeader, ".png") {
		t.Fatal("expected non-image content to be rejected")
	}
}

func TestValidImageContentAcceptsPNG(t *testing.T) {
	fileHeader := multipartHeader(t, "image.png", []byte{
		0x89, 0x50, 0x4e, 0x47, 0x0d, 0x0a, 0x1a, 0x0a,
		0x00, 0x00, 0x00, 0x0d,
	})

	if !validImageContent(fileHeader, ".png") {
		t.Fatal("expected PNG content to be accepted")
	}
}

func multipartHeader(t *testing.T, filename string, content []byte) *multipart.FileHeader {
	t.Helper()

	var body bytes.Buffer
	writer := multipart.NewWriter(&body)
	part, err := writer.CreateFormFile("file", filename)
	if err != nil {
		t.Fatalf("CreateFormFile: %v", err)
	}
	if _, err := part.Write(content); err != nil {
		t.Fatalf("write content: %v", err)
	}
	if err := writer.Close(); err != nil {
		t.Fatalf("close writer: %v", err)
	}

	reader := multipart.NewReader(&body, writer.Boundary())
	form, err := reader.ReadForm(1024)
	if err != nil {
		t.Fatalf("ReadForm: %v", err)
	}
	t.Cleanup(func() {
		_ = form.RemoveAll()
	})
	files := form.File["file"]
	if len(files) != 1 {
		t.Fatalf("expected one file, got %d", len(files))
	}
	return files[0]
}
