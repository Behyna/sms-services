package service

func NewServiceError(code string, cause error) error {
	return Error{
		Code:  code,
		Cause: cause,
	}
}

type Error struct {
	Code  string
	Cause error
}

func (e Error) Error() string {
	return e.Cause.Error()
}

func (e Error) Unwrap() error {
	return e.Cause
}
