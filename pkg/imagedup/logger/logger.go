package logger

import (
	"bytes"
	"encoding/json"
	"fmt"
	"os"
	"strings"

	"github.com/kmulvey/imagedup/v2/internal/app/imagedup/hash"
	"github.com/sirupsen/logrus"
	log "github.com/sirupsen/logrus"
)

// DeleteLogger is a custom sirupsen/logrus logger to format output pairs as json
type DeleteLogger struct {
	LogFile *os.File
	Logrus  *logrus.Logger
}

// Format is a custom Logrus formatter that satisifes the Format interface
func (f *DeleteLogger) Format(entry *logrus.Entry) ([]byte, error) {
	var buf = new(bytes.Buffer)
	buf.WriteString(fmt.Sprintf("\n%s: %s		", "big", entry.Data["big"].(string)))
	buf.WriteString(fmt.Sprintf("%s: %s\n", "small", entry.Data["small"].(string)))

	var js, err = json.Marshal(entry.Data)
	if err != nil {
		var dataStr strings.Builder
		dataStr.WriteString("[")
		for k, v := range entry.Data {
			dataStr.WriteString(fmt.Sprintf("key: %s, val: %s; ", k, v))
		}
		dataStr.WriteString("]")
		return nil, fmt.Errorf("DeleteLogger could not marshal json data: %s, err: %w", dataStr.String(), err)
	}
	return append(js, '\n'), nil
}

// NewDeleteLogger is a convience output logger for imagedup and is compatible with the verify tool.
func NewDeleteLogger(filename string) (*DeleteLogger, error) {
	var file, err = os.OpenFile(filename, os.O_RDWR|os.O_CREATE, 0755)
	if err != nil {
		return nil, fmt.Errorf("DeleteLogger could not open file: %s, err: %w", filename, err)
	}

	var deleteLogger = logrus.New()
	deleteLogger.SetFormatter(new(DeleteLogger))
	deleteLogger.SetOutput(file)

	return &DeleteLogger{LogFile: file, Logrus: deleteLogger}, nil
}

// LogResult logs a single result as json
func (dl *DeleteLogger) LogResult(result hash.DiffResult) {
	if result.OneArea > result.TwoArea {
		dl.Logrus.WithFields(log.Fields{
			"big":   result.One,
			"small": result.Two,
		}).Info("delete")
	} else {
		dl.Logrus.WithFields(log.Fields{
			"big":   result.Two,
			"small": result.One,
		}).Info("delete")
	}
}

// Close the log file; mainly used in testing
func (dl *DeleteLogger) Close() error {
	return dl.LogFile.Close()
}
