package imagedup

import (
	"bytes"
	"encoding/json"
	"fmt"
	"os"
	"strings"

	"github.com/sirupsen/logrus"
)

type deleteLogFormatter struct{}

func (f *deleteLogFormatter) Format(entry *logrus.Entry) ([]byte, error) {
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

func newDeleteLogger(filename string) (*logrus.Logger, error) {
	var file, err = os.OpenFile(filename, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0666)
	if err != nil {
		return nil, fmt.Errorf("DeleteLogger could not open file: %s, err: %w", filename, err)
	}

	var deleteLogger = logrus.New()
	deleteLogger.SetFormatter(new(deleteLogFormatter))
	deleteLogger.SetOutput(file)

	return deleteLogger, nil
}
