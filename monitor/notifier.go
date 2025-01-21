package monitor

type Notifier interface {
	Send(title, text string) error
}
