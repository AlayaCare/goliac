package observability

type Warning error

type ErrorCollection struct {
	Errors []error
	Warns  []Warning
}

func (ec *ErrorCollection) AddError(err error) {
	ec.Errors = append(ec.Errors, err)
}

func (ec *ErrorCollection) AddWarn(err Warning) {
	ec.Warns = append(ec.Warns, err)
}

func (ec *ErrorCollection) HasErrors() bool {
	return len(ec.Errors) > 0
}

func (ec *ErrorCollection) HasWarns() bool {
	return len(ec.Warns) > 0
}

func NewErrorCollection() *ErrorCollection {
	return &ErrorCollection{
		Errors: []error{},
		Warns:  []Warning{},
	}
}
