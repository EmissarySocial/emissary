package ascacherules

// The ladder below is a complete unit table, in seconds.  Entries that no rule
// currently references are kept so a new rule never has to re-derive one.

// second represents the number of seconds in a second
//
//lint:ignore U1000 kept so the unit ladder below is complete
const second = 1

// minute represents the number of seconds in a minute
const minute = 60

// hour represents the number of seconds in an hour
const hour = 60 * 60

// day represents the number of seconds in a day
const day = 60 * 60 * 24

// week represents the number of seconds in a week
//
//lint:ignore U1000 kept so the unit ladder above is complete
const week = 60 * 60 * 24 * 7

// month represents the number of seconds in 30 days.  No, not exactly a month, but close enough for ascache
const month = 60 * 60 * 24 * 30

// year represents the number of seconds in 365 days.  No, not exactly a year, but close enough for ascache
const year = 60 * 60 * 24 * 365
