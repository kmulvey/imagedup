package logger

import (
	"bytes"
	"encoding/json"
	"fmt"
	"os"

	"github.com/sirupsen/logrus"
)

type DeleteLogFormatter struct{}

func (f *DeleteLogFormatter) Format(entry *logrus.Entry) ([]byte, error) {
	var buf = new(bytes.Buffer)
	buf.WriteString(fmt.Sprintf("\n%s: %s		", "big", entry.Data["big"].(string)))
	buf.WriteString(fmt.Sprintf("%s: %s\n", "small", entry.Data["small"].(string)))

	var js, _ = json.Marshal(entry.Data)
	return append(js, '\n'), nil
}

func NewDeleteLogger(filename string) (*logrus.Logger, error) {
	var file, err = os.OpenFile(filename, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0666)
	if err != nil {
		return nil, fmt.Errorf("DeleteLogger could not open file: %s, err: %w", filename, err)
	}

	var deleteLogger = logrus.New()
	deleteLogger.SetFormatter(new(DeleteLogFormatter))
	deleteLogger.SetOutput(file)

	return deleteLogger, nil
}
