behavior AsModal()

    init
        put "body" into [@hx-target]
        put "beforeend" into [@hx-swap]
        put "false" into [@hx-push-url]
        put "true" into [@data-preload]

        call htmx.process(me)
    end
end

on closeModal(event)
    if #modal is undefined then return end
    add .closing to #modal
    set window.location to event.detail.nextPage unless no event.detail.nextPage
    settle
    remove #modal
