package ascache

// CalcRefreshDate determines the date that a document should be refreshed,
// which is half the duration between the load time and the expiration time.
// At a minimum, refresh duration will not be any less than one day.
func CalcRefreshDate(loadDate int64, expirationDate int64) int64 {

	const oneDay = int64(60 * 60 * 24)

	refreshDuration := (expirationDate - loadDate) / 2

	if refreshDuration < oneDay {
		refreshDuration = oneDay
	}

	return loadDate + refreshDuration
}
