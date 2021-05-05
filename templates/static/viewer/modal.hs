on click from .modal-underlay 
    log event
    log #modal
    send closeModal to #modal

on closeModal(event)
    add .closing to #modal
    set window.location to event.detail.nextPage unless no event.detail.nextPage
    settle
    remove #modal
