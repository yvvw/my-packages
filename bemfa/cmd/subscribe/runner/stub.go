package runner

type stubRunner struct {
	topic string
}

func (a *stubRunner) Topic() string {
	return a.topic
}
