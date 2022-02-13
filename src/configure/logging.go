package configure

import (
	"os"
	"time"

	"github.com/sirupsen/logrus"
)

func init() {
	// log.SetOutput(io.Discard)
}

func initLogging(level string) {
	formatter := &logrus.TextFormatter{
		DisableColors:    true,
		ForceQuote:       true,
		FullTimestamp:    true,
		QuoteEmptyFields: true,
		TimestampFormat:  time.RFC3339,
		PadLevelText:     true,
	}

	logrus.SetFormatter(formatter)

	if lvl, err := logrus.ParseLevel(level); err == nil {
		logrus.SetLevel(lvl)
		if lvl >= logrus.DebugLevel {
			logrus.SetReportCaller(true)
		}
	} else {
		logrus.Warn("bad level: ", err)
	}

	logrus.SetOutput(os.Stdout)
}
