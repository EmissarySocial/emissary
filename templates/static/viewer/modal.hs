behavior AsModal(url)

	init
		put "body" into [@hx-target]
		put "beforeend" into [@hx-swap]
		put "false" into [@hx-push-url]
		put "true" into [@data-preload]

		call htmx.process(me)
	end
end

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
