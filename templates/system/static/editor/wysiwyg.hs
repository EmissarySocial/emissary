behavior wysiwyg(id)

    init 
        add .wysiwyg
        add [@tabIndex=0]
        add [@contentEditable=true]
        log "wysiwyg behavior:" + id

    on keydown(key, ctrlKey, metaKey)
        if ctrlKey or metaKey then 
            if key == "b" then 
                log "bold"
                halt
            end
            if key == "i" then 
                log "italics"
                halt
            end
        end

    -- Save on blur, or every 10 seconds
    on blur or keyup debounced at 10s
        set node to document.getElementById(id)
        if node == null then
            log "error, node: " + id + " is undefined."
            exit
        end

        set the node's value to my innerHTML
        trigger save