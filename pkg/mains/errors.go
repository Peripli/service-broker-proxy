package main

import (
	"fmt"

	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"
)

func func1() error {
	return errors.New("error from func1")
}

func func2() error {
	return errors.Wrap(func1(), "wrapper")
}

func func3() error {
	return errors.New("err bro")
}

func main() {

	//logging := newLogger(logrus.DebugLevel, &httputils.ErrorLocationHook{}, &httputils.LogLocationHook{})
	err := errors.WithStack(func3())
	//logging.WithError(err).Errorf("error logging")
	errorf := fmt.Errorf("%+v", err)
	logrus.Error(errorf)
	//err := func2()
	//fmt.Printf("%+v", err)

	//logging.WithError(err).Infof("hello brozz")
}

func newLogger(level logrus.Level, hooks ...logrus.Hook) *logrus.Logger {
	logger := logrus.New()
	logger.Level = level

	for _, hook := range hooks {
		logger.Hooks.Add(hook)
	}

	return logger
}
