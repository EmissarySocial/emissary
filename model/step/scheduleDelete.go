package step

import (
	"text/template"

	"github.com/benpate/derp"
	"github.com/benpate/rosetta/mapof"
)

// ScheduleDelete is a Step that can update a stream's ScheduleDeleteDate with the current time.
type ScheduleDelete struct {
	Days    *template.Template
	Hours   *template.Template
	Minutes *template.Template
	Seconds *template.Template
}

// NewScheduleDelete returns a fully initialized ScheduleDelete object
func NewScheduleDelete(stepInfo mapof.Any) (ScheduleDelete, error) {

	const location = "model.step.NewScheduleDelete"

	// Days template
	days, err := template.New("").Parse(stepInfo.GetString("days"))

	if err != nil {
		return ScheduleDelete{}, derp.Wrap(err, location, "Unable to parse 'days' template", stepInfo)
	}

	// Hours template
	hours, err := template.New("").Parse(stepInfo.GetString("hours"))

	if err != nil {
		return ScheduleDelete{}, derp.Wrap(err, location, "Unable to parse 'hours' template", stepInfo)
	}

	// Minutes template
	minutes, err := template.New("").Parse(stepInfo.GetString("minutes"))

	if err != nil {
		return ScheduleDelete{}, derp.Wrap(err, location, "Unable to parse 'minutes' template", stepInfo)
	}

	// Seconds template
	seconds, err := template.New("").Parse(stepInfo.GetString("seconds"))

	if err != nil {
		return ScheduleDelete{}, derp.Wrap(err, location, "Unable to parse 'seconds' template", stepInfo)
	}

	// Return the step
	result := ScheduleDelete{
		Days:    days,
		Hours:   hours,
		Minutes: minutes,
		Seconds: seconds,
	}

	return result, nil
}

// Name returns the name of the step, which is used in debugging.
func (step ScheduleDelete) Name() string {
	return "schedule-delete"
}

// RequiredModel returns the name of the model object that MUST be present in the Template.
// If this value is not empty, then the Template MUST use this model object.
func (step ScheduleDelete) RequiredModel() string {
	return "Stream"
}

// RequiredStates returns a slice of states that must be defined any Template that uses this Step
func (step ScheduleDelete) RequiredStates() []string {
	return []string{}
}

// RequiredRoles returns a slice of roles that must be defined any Template that uses this Step
func (step ScheduleDelete) RequiredRoles() []string {
	return []string{}
}
