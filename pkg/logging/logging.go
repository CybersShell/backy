package logging

type Logging struct {
	Err    error
	Output string
}

func logger() Logging {
	return Logging{}
}
