package derp

// Report takes ANY error (hopefully a derp error) and attempts to report it
// via all configured error reporting mechanisms.
func Report(err error) error {

	// If the error is nil, then there's nothing to do.
	if isNil(err) {
		return nil
	}

	for _, plugin := range Plugins {
		plugin.Report(err)
	}

	return err
}
