behavior PrettyDate(date)

	on load

		repeat until done

			make a Date from (Date.now()) called now
			set miliseconds to (now - original)
			set seconds to Math.floor(miliseconds / 1000)
			set minutes to Math.floor(seconds / 60)

			if minutes == 0 then 

				if seconds < 30 then
					set my innerHTML to "a few seconds ago"
					return
					-- wait ((30 * 1000) - miliseconds) ms -- wait until 30-second mark
				else
					set my innerHTML to "30 seconds ago"
					return
					-- wait ((60 * 1000) - miliseconds) ms -- wait until 60-second mark
				end

			else
				set my innerHTML to "more than 1 minute"
				return
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