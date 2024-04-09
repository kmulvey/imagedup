package logger

import (
	"encoding/json"
	"fmt"
	"os"

	"github.com/kmulvey/imagedup/v2/internal/app/imagedup/hash"
)

// DeleteLogger logs the duplicate file pairs and can read with the verify tool.
type DeleteLogger struct {
	LogFile *os.File
}

// DeleteEntry is a duplicate file pair
type DeleteEntry struct {
	Big   string
	Small string
}

// NewDeleteLogger creates a new DeleteLogger and deletes the log file if it already exists.
func NewDeleteLogger(filename string) (*DeleteLogger, error) {

	// delete existing file
	if _, err := os.Stat(filename); err == nil {
		err = os.RemoveAll(filename)
		if err != nil {
			return nil, fmt.Errorf("DeleteLogger was unable to remove existsing delete file: %s, err: %w", filename, err)
		}
	}

	var file, err = os.OpenFile(filename, os.O_RDWR|os.O_CREATE, 0755)
	if err != nil {
		return nil, fmt.Errorf("DeleteLogger could not open file: %s, err: %w", filename, err)
	}

	// write the header
	file.WriteString("[")

	return &DeleteLogger{LogFile: file}, nil
}

// ReadDeleteLogFile reads the entire file and returns a slice of DeleteEntries.
func ReadDeleteLogFile(filename string) ([]DeleteEntry, error) {

	var content, err = os.ReadFile(filename)
	if err != nil {
		return nil, fmt.Errorf("DeleteLogger could not open file: %s, err: %w", filename, err)
	}

	var entries []DeleteEntry
	err = json.Unmarshal(content, &entries)
	if err != nil {
		return nil, fmt.Errorf("DeleteLogger could unmarshal file to []DeleteEntry, file: %s, err: %w", filename, err)
	}

	return entries, nil
}

// LogResult logs a single duplicate result as json.
func (dl *DeleteLogger) LogResult(result hash.DiffResult) error {
	var entry DeleteEntry

	if result.OneArea > result.TwoArea {
		entry = DeleteEntry{
			Big:   result.One,
			Small: result.Two,
		}
	} else {
		entry = DeleteEntry{
			Big:   result.Two,
			Small: result.One,
		}
	}

	var js, err = json.Marshal(entry)
	if err != nil {
		return fmt.Errorf("DeleteLogger could not marshal DiffResult json, err: %w", err)
	}
	_, err = dl.LogFile.Write(js)
	return err
}

// Close writes the trailing ] and closes the log file
func (dl *DeleteLogger) Close() error {
	dl.LogFile.WriteString("]")
	return dl.LogFile.Close()
}
