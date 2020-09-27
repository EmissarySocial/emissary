package render

type FormWrapper struct {
	templateService TemplateService
	stream          StreamWrapper
	Transition      string
}

func (w FormWrapper) Render() (string, error) {
	return "", nil
}

func (w FormWrapper) Stream() StreamWrapper {
	return w.stream
}
