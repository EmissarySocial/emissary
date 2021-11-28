behavior AsModal(url)

    init
        put "body" into [@hx-target]
        put "beforeend" into [@hx-swap]
        put "false" into [@hx-push-url]
        put "true" into [@data-preload]

        call htmx.process(me)
    end
end

on closeModal(event)
    if #modal is not empty then 
        add .closing to #modal
        set window.location to event.detail.nextPage unless no event.detail.nextPage
        settle
        remove #modal
    end
end
