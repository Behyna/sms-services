package mq

type TempError struct {
	Err error
}

func (e TempError) Error() string {
	return e.Err.Error()
}

func (e TempError) Temporary() bool {
	return true
}

func Temporary(err error) error {
	return TempError{Err: err}
}
