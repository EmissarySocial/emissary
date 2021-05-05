eventsource StreamServer from /{{.Token}}/sse
        on message
            for el in query(`[stream="${it}"]`)
                send stream:update to it
            end
        end
    end

    on htmx:pushedIntoHistory or htmx:historyRestore
        call StreamServer.open(window.location.pathname + "/sse")
    end
end