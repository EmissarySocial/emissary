"use strict";
(() => {
  // src/service/activityPub.ts
  var ActivityPubService = class {
    // All class #properties are PRIVATE
    #actorID = "";
    async start(actorID) {
      this.#actorID = actorID;
    }
    createObject(object) {
      return this.sendActivity({
        "@context": "",
        "id": "",
        "type": "Create",
        "actor": this.#actorID,
        "object": object
      });
    }
    deleteObject(objectId) {
      return this.sendActivity({
        "@context": "",
        "id": "",
        "type": "Delete",
        "actor": this.#actorID,
        "object": objectId
      });
    }
    sendActivity(activity) {
      try {
        fetch("/@me/outbox", {
          method: "POST",
          body: JSON.stringify(activity)
        });
        return true;
      } catch (err) {
        console.log(err);
        return false;
      }
    }
  };
})();
