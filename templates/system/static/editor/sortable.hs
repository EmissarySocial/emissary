behavior NebulaLayout
        
    init
        set layout to first <.nebula-layout /> in me
        set options to {
            sort: true,
            draggable: ".nebula-layout-item",
            handle: ".nebula-layout-sortable-handle",
            ghostClass: "nebula-layout-sortable-ghost",
            direction: "vertical"
        }

        make a Sortable from layout, options called sortable
    end

    on end
        set layout to first <.nebula-layout /> in me
        set childIds to []
        for item in (<.nebula-layout-item /> in me)
            append item's [@data-id] to childIds
        end

        set body to {
            "action":"sort-children",
            "itemId": layout's [@data-id],
            "childIds": childIds,
            "check": layout's [@data-check]
        } as FormEncoded

        fetch `${document.location.pathname}` with 
            method:"POST", 
            headers: {"content-type":"application/x-www-form-urlencoded"},
            body:body

        log it
    end
end