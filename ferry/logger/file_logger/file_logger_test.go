package file_logger

import "testing"

func TestFileLogger(t *testing.T) {
	fileLogger, err := NewFileLogger()
	if err != nil {
		t.Fatal(err)
	}
	defer fileLogger.Close()

	fileLogger.GetLogger().Println("Hello there bye")
}
