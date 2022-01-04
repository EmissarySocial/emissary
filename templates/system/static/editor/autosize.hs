behavior Autosize 

    init
        send autosize

    on input
        send autosize

    on autosize
        if my scrollTop != 0
            set my style.height to my scrollHeight + "px"
        end
    end
end
