package file_logger

import (
	"log"
	"os"
	"path/filepath"
)

// file logger
type FileLogger struct {
	log  *log.Logger
	file *os.File
}

func NewFileLogger() (*FileLogger, error) {
	path := filepath.Join("..", "..", "logs", "app.log")
	if err := os.MkdirAll(filepath.Dir(path), 0755); err != nil {
		return nil, err
	}

	file, err := os.OpenFile(path, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
	if err != nil {
		return nil, err
	}

	fileLogger := log.New(file, "", log.LstdFlags|log.Lshortfile)

	return &FileLogger{
		log:  fileLogger,
		file: file,
	}, nil
}

func (f *FileLogger) GetLogger() *log.Logger {
	return f.log
}

func (f *FileLogger) Close() error {
	return f.file.Close()
}
