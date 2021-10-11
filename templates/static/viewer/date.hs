behavior PrettyDate(date)

	on load
		make a Date from (date * 1000) called original
		set done to false

		repeat until done

			make a Date from (Date.now()) called now
			set miliseconds to (now - original)
			set seconds to Math.floor(miliseconds / 1000)
			set minutes to Math.floor(seconds / 60)

			if minutes == 0 then 

				if seconds < 30 then
					set my innerHTML to "a few seconds ago"
					wait ((30 * 1000) - miliseconds) ms -- wait until 30-second mark
				else
					set my innerHTML to "30 seconds ago"
					wait ((60 * 1000) - miliseconds) ms -- wait until 60-second mark
				end

			else

				set hours to Math.floor(minutes / 60)

				if hours == 0 then 
					set my innerHTML to pluralize(minutes, "minute")
					wait (((minutes + 1) * 60 * 1000) - miliseconds) ms -- wait until next minute turns
				else
					
					set days to Math.floor(hours / 24)

					if days == 0 then
						set my innerHTML to pluralize(hours, "hour")
						set done to true
					else

						set months to monthDiff(original, now)

						if months == 0 then
							set my innerHTML to pluralize(days, "day")
							set done to true
						else 
							
							set years to Math.floor(months / 12)
							
							if years == 0 then 
								set my innerHTML to pluralize(months, "month")
								set done to true
							else
								set my innerHTML to pluralize(years, "year")
								set done to true
							end -- if years

						end -- if months

					end -- if days

				end -- if hours

			end -- if minutes

		end -- repeat

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