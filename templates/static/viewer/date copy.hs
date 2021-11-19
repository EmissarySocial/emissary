behavior PrettyDate(date)

	on load
		make a Date from (date * 1000) called original

		repeat forever

			make a Date from (Date.now()) called now
			set miliseconds to (now - original)
			set seconds to Math.floor(miliseconds / 1000)

			-- SECONDS
			if seconds < 30 then
				set my innerHTML to "a few seconds ago"
				wait ((30 * 1000) - miliseconds) ms -- wait until 30-second mark
				continue
			end

			if seconds < 60 then 
				set my innerHTML to "30 seconds ago"
				wait ((60 * 1000) - miliseconds) ms -- wait until 60-second mark
				continue
			end

			-- MINUTES
			set minutes to Math.floor(seconds / 60)

			if minutes < 60 then
				set my innerHTML to pluralize(minutes, "minute")
				wait (((minutes + 1) * 60 * 1000) - miliseconds) ms -- wait until next minute turns
				continue
			end

			-- HOURS
			set hours to Math.floor(minutes / 60)

			if hours < 24 then 
				set my innerHTML to pluralize(hours, "hour")
				return
			end

			-- DAYS + MONTHS
			set days to Math.floor(hours / 24)
			set months to monthDiff(original, now)

			if months == 0 then
				set my innerHTML to pluralize(days, "day")
				return
			end

			if months < 12 then
				set my innerHTML to pluralize(months, "month")
				return
			end

			-- YEARS
			set years to Math.floor(months / 12)						
			set my innerHTML to pluralize(years, "year")
			return

		end -- repeat if no return

	end -- on load

end -- behavior


def pluralize(number, unit)
	set result to number + " " + unit

	if number > 1
		append "s" to result
	end

	append (" ago") to result
	return result
end

def monthDiff(a, b)
	set years to b.getFullYear() - a.getFullYear()
	return (years * 12) + (b.getMonth() - a.getMonth())
end