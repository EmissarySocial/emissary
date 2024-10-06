package queue

type Consumer interface {
	Name() string
	Run(Task) error
}
