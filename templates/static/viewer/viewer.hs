-- init 
-- log (window.location + "/sse")
-- call StreamServer.connect(window.location + "/sse")

eventsource StreamServer

    on message
        for el in <[stream=`${it}`]/>
            send stream:update to it
        end
    end
