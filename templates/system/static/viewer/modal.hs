
on closeModal(nextPage)

	if #modal is not empty then 
		add .closing to #modal
		settle
		remove #modal
	end

	if nextPage is not empty then 
		set the window's location to nextPage
	end
end

on keypress[key=="Escape"] from window
	if #modal is not empty then 
		trigger closeModal
		halt
	end
end

behavior AsModal

	init
		put "aside" into [@hx-target]
		put "innerhtml" into [@hx-swap]
		put "false" into [@hx-push-url]
		put "true" into [@data-preload]

		call htmx.process(me)
	end
end

behavior ModalCancelButton() 

	on click 
		send closeModal to #modal
	end
end
