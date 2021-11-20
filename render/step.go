package render

type Step interface {
	Get(*Renderer) error
	Post(*Renderer) error
}
