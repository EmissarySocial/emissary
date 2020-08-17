package service

import "github.com/benpate/ghost/model"

type Render struct {
	factory Factory
}

type Renderer interface {
	Execute() (string, error)
	Into(Renderer) Renderer
}

func (render Render) Pipeline(renderers ...Renderer) Renderer {
	return nil
}

func (render Render) Stream(stream *model.Stream) Renderer {
	return nil
}

func (render Render) Form(form *model.Stream, transition *model.Transition) Renderer {
	return nil
}

func (render Render) Page(listen string) Renderer {
	return nil
}

func (render Render) Global() Renderer {
	return nil
}
