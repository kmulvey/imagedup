package logger

import (
	"encoding/json"
	"fmt"
	"os"

	"github.com/kmulvey/imagedup/v2/internal/app/imagedup/hash"
)

// DeleteLogger logs the duplicate file pairs and can read with the verify tool.
type DeleteLogger struct {
	FileName   string
	LogFile    *os.File
	FirstEntry bool // used to tell if we should write a ',' after the entry
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

	return &DeleteLogger{FileName: filename, LogFile: file, FirstEntry: true}, nil
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

// LogResult logs a single duplicate result as json. Each record is writted to disk immediatly as to not use too much RAM.
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
		return fmt.Errorf("DeleteLogger could not marshal DiffResult json, file: %s, err: %w", dl.FileName, err)
	}

	if !dl.FirstEntry {
		_, err = dl.LogFile.WriteString(",")
		if err != nil {
			return fmt.Errorf("DeleteLogger could not write the comma, file: %s, err: %w", dl.FileName, err)
		}
	}

	_, err = dl.LogFile.Write(js)
	if err != nil {
		return fmt.Errorf("DeleteLogger could not write the JSON to the file: %s, err: %w", dl.FileName, err)
	}

	dl.FirstEntry = false
	return nil
}

// Close writes the trailing ] and closes the log file.
func (dl *DeleteLogger) Close() error {
	var _, err = dl.LogFile.WriteString("]")
	if err != nil {
		return fmt.Errorf("DeleteLogger could not write the trailing ] to the file: %s, err: %w", dl.FileName, err)
	}

	err = dl.LogFile.Close()
	if err != nil {
		return fmt.Errorf("DeleteLogger could not close the file: %s, err: %w", dl.FileName, err)
	}

	return nil
}
