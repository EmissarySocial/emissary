-- handles keyboard shortcuts
on keypress(keyCode, metaKey, ctrlKey) from [data-shortcut]
    
    if ctrlKey or metaKey then
        log me
        log ctrlKey
        log metaKey
        log keyCode
    end
end