behavior PrettyDate(date)

	on load

		if date == 0 then 
			exit
		end

		repeat forever 
			make a Date from (date) called original
			make a Date from (Date.now()) called now
			
			set milisecondCount to (now - original)
			set secondCount to Math.floor(milisecondCount / 1000)
		
			if secondCount < 60 then
				set my innerHTML to "just now"
				wait ((60 * 1000) - milisecondCount) ms 
				continue
			end
			set minuteCount to Math.floor(secondCount / 60)

			if minuteCount < 60 then 
				set my innerHTML to minuteCount + "min ago"
				wait((60 * 60 * 1000) - milisecondCount) ms
				continue
			end

			set hourCount to Math.floor(minuteCount / 60)

			if hourCount < 24 then
				set my innerHTML to hourCount + "h ago"
				exit
			end

			set dayCount to Math.floor(hourCount / 24)
			set monthCount to DateDiffMonths(original, now)
			set yearCount to DateDiffYears(original, now)

			if yearCount > 0 then 
				set my innerHTML to yearCount + "y ago"
				exit
			end

			if monthCount >= 2 then 
				set my innerHTML to monthCount + "m ago"
			end

			set my innerHTML to dayCount + "d ago"

		end
	end
end

def DateDiffMonths(old, new)
	set oldYear to old.getYear()
	set oldMonth to old.getMonth()
	set newYear to new.getYear()
	set newMonth to new.getMonth()
	set result to (newYear - oldYear) * 12

	if oldMonth > newMonth then
		set result to result - 1
	end

	return result


def DateDiffYears(old, new) 

	set oldYear to old.getYear()
	set oldMonth to old.getMonth()
	set newYear to new.getYear()
	set newMonth to new.getMonth()
	set result to newYear - oldYear

	if oldMonth > newMonth then
		set result to result - 1
	end

	return result


