
def pluralizeTime(number, unit)
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


behavior PrettyDate(date)

	on load

		log "behavior"
		log my innerHTML

		-- repeat until done

			make a Date from (Date.now()) called now
			set miliseconds to (now - date)
			set seconds to Math.floor(miliseconds / 1000)
			log seconds

			if seconds < 30 then
				log "A?"
				set my innerHTML to "a few seconds ago"
				return
			end
			log "here?"

			if seconds < 60 then 
				set my innerHTML to "30 seconds ago"
				return
			end


			set minutes to Math.floor(seconds / 60)
			log minutes

			if minutes < 60 then 
				set my innerHTML to pluralizeTime(minutes, "minute")
				return
			end

			set hours to Math.floor(minutes / 60)
			log hours

			if hours < 24 then
				set my innerHTML to pluralizeTime(hours, "hour")
				return
			end

			set days to Math.floor(hours / 24)
			log days

			if days < 7 then 
				set my innerHTML to pluralizeTime(days, "day")
				return
			end

			set weeks to Math.floor(days / 7)
			log weeks

			if weeks < 8 then 
				set my innerHTML to pluralizeTime(weeks, "week")
			end

			set my innerHTML to "very long (" + pluralizeTime(days, "day") + ")"
		-- end

	end

end

