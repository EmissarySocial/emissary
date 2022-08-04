package queue

type Task interface {
	Run() error
}
