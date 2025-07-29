package main

import (
	"fmt"
	"image"
	"image/png"
	"os"
)

type FileOutputHandler struct {
	filePath string
}

func NewFileOutputHandler(filePath string) *FileOutputHandler {
	return &FileOutputHandler{
		filePath: filePath,
	}
}

func (f *FileOutputHandler) GetType() string {
	return "file"
}

func (f *FileOutputHandler) Output(img image.Image) error {
	file, err := os.Create(f.filePath)
	if err != nil {
		return fmt.Errorf("failed to create file: %v", err)
	}
	defer file.Close()

	return png.Encode(file, img)
}

func (f *FileOutputHandler) Close() error {
	return nil
}
