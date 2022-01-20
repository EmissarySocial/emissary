
behavior TabContainer

	-- init handles the default tab selection.  If a document hash exists 
	-- (and points to one of our tabs) then select it first.  Otherwise,
	-- select the first tab in the list
	init
		if window.location.hash is not "" then
			set target to the first <[aria-controls="${window.location.hash.slice(1)}"]/>
		end

		if target is null then 
			set target to first <[role=tab]/> in me
		end

		send select to target
	end

	-- move to first tab on HOME
	on keydown[keyCode==36]
		send select to first <[role=tab]/> in me
		halt the event

	-- move to last tab on END
	on keydown[keyCode==35]
		send select to last <[role=tab]/> in me
		halt the event

	-- move to previous tab LEFT ARROW
	-- on keydown[keyCode==37]
	--	set current to first <[aria-selected=true]/> in me
	--	send select to previous <[role=tab]/> from current with wrapping
	--	halt the event
	
	-- move to next tab on RIGHT ARROW
	-- on keydown[keyCode==39]
	--	set current to first <[aria-selected=true]/> in me
	--	send select to next <[role=tab]/> from current with wrapping
	--	halt the event
	
	-- select highlighted tab on SPACE (expected for ARIA buttons)
	on keydown[keyCode==32]
		send select to first <[role=tab]:focus/>
		halt the event

	-- select highlighted tab on ENTER (additional key to select tabs)
	-- on keydown[keyCode==13]
	--	send select to first <[role=tab]:focus/>
	--	halt the event

	-- handle mouse clicks directly on tabs
	on mousedown(target)[button==0] from <[role=tablist] [role=tab] />
		send select to target
		halt the event
	end

	on click from <[role=tablist] [role=tab] />
		halt the event

	-- handle touch events for phones and tablets
	on touchstart(target) from <[role=tablist] [role=tab] />
		send select to target
		halt the event
	end

	on select(target)
		for tab in <[role=tab] /> in me
			if tab == target
				add [@aria-selected="true"] to tab
				call window.history.replaceState(undefined, tab.innerHTML, "#" + target[@aria-controls])
			else
				remove [@aria-selected] from tab
			end
		end

		for panel in <[role=tabpanel] /> in me
			set {hidden: (panel[@id] != target[@aria-controls])} on panel
		end
	end
end

