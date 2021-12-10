behavior PrettyDate(date)

	on load

		if date == 0 then 
			exit
		end

		make a Date from (date) called original
		set my innerHTML to original
		exit

		repeat forever

			make a Date from (date) called original
			make a Date from (Date.now()) called now

			set miliseconds to (now - original)
			set seconds to Math.floor(miliseconds / 1000)
			
			if seconds < 30 then
				set my innerHTML to "just now"
				-- wait until 30-second mark
				wait ((30 * 1000) - miliseconds) ms 
				continue
			end
		
			if seconds < 60 then 
				set my innerHTML to "30 seconds ago"
				-- wait until 30-second mark
				wait ((60 * 1000) - miliseconds) ms 
				continue
			end

			set minutes to Math.floor(seconds / 60)

			if minutes < 60 then 
				set my innerHTML to PluralizeTime(minutes, "minute")
				-- wait until next minute turns
				set timeout to (((minutes + 1) * 60 * 1000) - miliseconds)
				log "minutes.."
				log (minutes + 1) * 60 * 1000
				log timeout
				exit 
				wait timeout ms 
				continue
			end

			exit

			log minutes
			set my innerHTML to "at least a minute"
			exit

			set hours to Math.floor(minutes / 60)

			if hours < 24 then
				set my innerHTML to PluralizeTime(hours, "hour")
				exit
			end

			set days to Math.floor(hours / 24)

			if days < 7 then 
				set my innerHTML to PluralizeTime(days, "day")
				exit
			end

			set weeks to Math.floor(days / 7)

			if weeks < 8 then 
				set my innerHTML to PluralizeTime(weeks, "week")
				exit
			end

			set months to MonthDiff(date, now)
			if months < 24 then 
				set my innerHTML to PluralizeTime(months, "month")
				exit
			end

			set years to YearDiff(date, now)
			set my innerHTML to PluralizeTime(years, "year")
			exit

		end
	end

def MonthDiff(old, new)
	set oldYear to old.getYear()
	set oldMonth to old.getMonth()
	set newYear to new.getYear()
	set newMonth to new.getMonth()
	set result to (newYear - oldYear) * 12

	if oldMonth > newMonth then
		set result to result - 1
	end

	return result


def YearDiff(old, new) 

	set oldYear to old.getYear()
	set oldMonth to old.getMonth()
	set newYear to new.getYear()
	set newMonth to new.getMonth()
	set result to newYear - oldYear

	if oldMonth > newMonth then
		set result to result - 1
	end

	return result


def PluralizeTime(number, unit)
	set result to number + " " + unit

	if number > 1
		append "s" to result
	end

	append (" ago") to result

	log result
	return result
end

