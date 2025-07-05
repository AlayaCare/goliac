package observability

import "github.com/sirupsen/logrus"

type Warning error

type InfoEntry struct {
	LogLevel logrus.Level
	Format   string
	Args     []any
	Fields   map[string]any
}

/*
LogCollection is used to collect logs (debug/info/warning/error)
and to ship them to multiple target (std output, but also the UI)
*/
type LogCollection struct {
	Logs   []InfoEntry
	Errors []error
	Warns  []Warning
}

func (ec *LogCollection) AddDebug(fields map[string]any, format string, args ...any) {
	entry := InfoEntry{
		LogLevel: logrus.DebugLevel,
		Format:   format,
		Args:     args,
		Fields:   fields,
	}
	ec.Logs = append(ec.Logs, entry)
}

func (ec *LogCollection) AddInfo(fields map[string]any, format string, args ...any) {
	entry := InfoEntry{
		LogLevel: logrus.InfoLevel,
		Format:   format,
		Args:     args,
		Fields:   fields,
	}
	ec.Logs = append(ec.Logs, entry)
}

func (ec *LogCollection) AddError(err error) {
	ec.Errors = append(ec.Errors, err)
}

func (ec *LogCollection) AddWarn(err Warning) {
	ec.Warns = append(ec.Warns, err)
}

func (ec *LogCollection) HasErrors() bool {
	return len(ec.Errors) > 0
}

func (ec *LogCollection) HasWarns() bool {
	return len(ec.Warns) > 0
}

func NewLogCollection() *LogCollection {
	return &LogCollection{
		Errors: []error{},
		Warns:  []Warning{},
	}
}

func (ec *LogCollection) ResetWarnings() {
	ec.Warns = []Warning{}
}
