package main

import (
	"encoding/base64"
	"fmt"
	"image/png"
	"math/rand"
	"os"
	"testing"
	"time"
	"unicode/utf8"
)

func TestGenerateCode(t *testing.T) {
	code := GenerateCode(6)
	fmt.Println(code)
	if actual := utf8.RuneCountInString(code); actual != 6 {
		t.Fatalf("code length expected %d but got %d\n", 6, actual)
	}
}

func TestLoadFont(t *testing.T) {
	f, err := LoadFont("Blox2.ttf")
	if err != nil {
		t.Fatalf("loading font: %v\n", err)
	}
	fmt.Println(f)
}

func TestGenerateImage(t *testing.T) {
	rand.Seed(time.Now().UnixNano())
	os.Setenv("FONT", "Blox2.ttf")
	code := GenerateCode(6)
	fmt.Println(code)
	img, err := GenerateImage(code, 200, 100)
	if err != nil {
		t.Fatalf("generate image: %v\n", err)
	}
	file, err := os.CreateTemp(os.TempDir(), fmt.Sprintf("%s.png", time.Now().Format(time.RFC3339)))
	if err != nil {
		t.Fatalf("open temp file: %v\n", err)
	}
	defer file.Close()
	fmt.Println(file.Name())
	fmt.Println(len(base64.StdEncoding.EncodeToString(img.Pix)))

	png.Encode(file, img)
}

func TestListFiles(t *testing.T) {
	err := ListFiles()
	if err != nil {
		t.Fatalf("list files: %v\n", err)
	}
}
