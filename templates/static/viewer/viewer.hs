
init 

    call StreamServer.connect(window.location + "/sse")


eventsource StreamServer

    on message
        for el in <[stream=`${it}`]/>
            send stream:update to it
        end
    end

    on htmx:pushedIntoHistory or htmx:historyRestore
        call StreamServer.open(window.location.pathname + "/sse")
    end

