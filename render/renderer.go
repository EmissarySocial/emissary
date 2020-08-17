package render

type renderer interface {
	execute() (string, error)
}
