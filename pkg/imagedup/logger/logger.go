package logger

import (
	"bytes"
	"encoding/json"
	"fmt"
	"os"
	"strings"

	"github.com/kmulvey/imagedup/internal/app/imagedup/hash"
	"github.com/sirupsen/logrus"
	log "github.com/sirupsen/logrus"
)

type DeleteLogFormatter struct{}

func (f *DeleteLogFormatter) Format(entry *logrus.Entry) ([]byte, error) {
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

func NewDeleteLogger(filename string) (*logrus.Logger, error) {
	var file, err = os.Create(filename)
	if err != nil {
		return nil, fmt.Errorf("DeleteLogger could not open file: %s, err: %w", filename, err)
	}

	var deleteLogger = logrus.New()
	deleteLogger.SetFormatter(new(DeleteLogFormatter))
	deleteLogger.SetOutput(file)

	return deleteLogger, nil
}

func LogResults(resultsLogger *logrus.Logger, results chan hash.DiffResult) {
	for result := range results {
		if result.OneArea > result.TwoArea {
			resultsLogger.WithFields(log.Fields{
				"big":   result.One,
				"small": result.Two,
			}).Info("delete")
		} else {
			resultsLogger.WithFields(log.Fields{
				"big":   result.Two,
				"small": result.One,
			}).Info("delete")
		}
	}
}

func LogResult(resultsLogger *logrus.Logger, result hash.DiffResult) {
	if result.OneArea > result.TwoArea {
		resultsLogger.WithFields(log.Fields{
			"big":   result.One,
			"small": result.Two,
		}).Info("delete")
	} else {
		resultsLogger.WithFields(log.Fields{
			"big":   result.Two,
			"small": result.One,
		}).Info("delete")
	}
}
