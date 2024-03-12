package builder

type PipelineBehavior func(*PipelineResult)

// Continue is a NOOP that does not change the PipelineResult object
func Continue() PipelineBehavior {
	return func(_ *PipelineResult) {}
}

// Halt sets the Halt flag on the PipelineResult object
func Halt() PipelineBehavior {
	return func(status *PipelineResult) {
		status.Halt = true
	}
}

// UseResult takes a new PipelineResult object, and merges it into the existing PipelineResult object.
func UseResult(newStatus PipelineResult) PipelineBehavior {
	return func(oldStatus *PipelineResult) {
		oldStatus.Merge(newStatus)
	}
}

// AsFullPage sets the FullPage flag on the PipelineResult object, which
// tells the builder to NOT include the header/footer from the site theme.
func (exit PipelineBehavior) AsFullPage() PipelineBehavior {
	return func(status *PipelineResult) {
		if exit != nil {
			exit(status)
		}
		status.FullPage = true
	}
}

// WithError sets the Error value on the PipelineResult object
func (exit PipelineBehavior) WithError(err error) PipelineBehavior {
	return func(status *PipelineResult) {
		if exit != nil {
			exit(status)
		}
		status.Error = err
	}
}

// WithEvent adds an HX-Trigger event to the PipelineResult object
func (exit PipelineBehavior) WithEvent(name string, value string) PipelineBehavior {
	return func(status *PipelineResult) {
		if exit != nil {
			exit(status)
		}
		status.Events[name] = value
	}
}

// RemoveEvent removes an HX-Trigger event to the PipelineResult object
func (exit PipelineBehavior) RemoveEvent(name string) PipelineBehavior {
	return func(status *PipelineResult) {
		if exit != nil {
			exit(status)
		}
		delete(status.Events, name)
	}
}

// WithHeader adds an HX-Trigger event to the PipelineResult object
func (exit PipelineBehavior) WithHeader(name string, value string) PipelineBehavior {
	return func(status *PipelineResult) {
		if exit != nil {
			exit(status)
		}
		status.Headers[name] = value
	}
}

func (exit PipelineBehavior) WithStatusCode(statusCode int) PipelineBehavior {
	return func(status *PipelineResult) {
		if exit != nil {
			exit(status)
		}
		status.StatusCode = statusCode
	}
}

func (exit PipelineBehavior) WithContentType(contentType string) PipelineBehavior {
	return func(status *PipelineResult) {
		exit(status)
		status.ContentType = contentType
	}
}
