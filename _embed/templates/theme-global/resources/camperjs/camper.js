// src/utils.ts
function guessProtocol(server) {
  switch (server) {
    case "localhost":
    case "127.0.0.1":
      return "http://";
  }
  return "https://";
}
function hideElement(element, hide) {
  if (hide) {
    element.hidden = true;
    element.style.display = "none";
  } else {
    element.hidden = false;
    element.style.display = "";
  }
}

// src/webfinger.ts
var WebFinger = class {
  static getMetadata = async (username) => {
    const url = this.getUrl(username);
    if (url == null) {
      return null;
    }
    const response = await fetch(url);
    if (!response.ok) {
      console.error("WebFinger request failed with status " + response.status);
      return null;
    }
    const result = await response.json();
    return result;
  };
  // getActivityPubId retrieves user's ActivityPub Actor ID from WebFinger metadata
  static getActivityPubId = (webfingerResult) => {
    const links = webfingerResult.links || [];
    for (const link of links) {
      const relation = link.rel || "";
      if (relation.toLowerCase() == "self") {
        const linkType = link.type || "";
        if (linkType.toLowerCase() == "application/activity+json") {
          return link.href || "";
        }
      }
    }
    return "";
  };
  // getUrl constructs the well-known WebFinger URL to look up the provided username
  static getUrl = (username) => {
    const [user, server] = this.splitUsername(username);
    if (user == "" || server == "") {
      console.error("Invalid username: " + username);
      return null;
    }
    const result = guessProtocol(server) + server + "/.well-known/webfinger?resource=acct:" + user + "@" + server;
    return result;
  };
  // splitUsername splits a WebFinger username into its "user" and "server" parts
  static splitUsername = (username) => {
    if (username.startsWith("@")) {
      username = username.substring(1);
    }
    var parts = username.split("@");
    if (parts.length != 2) {
      console.error(username + " is not a valid username");
      return ["", ""];
    }
    return [parts[0], parts[1]];
  };
};

// src/nodeinfo.ts
var NodeInfo = class {
  static getSoftwareName = async (server) => {
    const nodeInfo = await this.getNodeInfo(server);
    if (nodeInfo == null) {
      return "";
    }
    return nodeInfo?.software?.name || "";
  };
  static getNodeInfo = async (server) => {
    const url = await this.#getNodeInfoUrl(server);
    if (url == null) {
      return null;
    }
    try {
      const response = await fetch(url);
      if (response.ok) {
        return await response.json();
      }
      console.error("NodeInfo request failed with status " + response.status);
    } catch (error) {
      console.error("NodeInfo request failed with error: " + error);
    }
    return null;
  };
  static #getNodeInfoUrl = async (server) => {
    try {
      const url = guessProtocol(server) + server + "/.well-known/nodeinfo";
      const response = await fetch(url);
      if (response.ok) {
        const result = await response.json();
        return result?.links.at(0)?.href || null;
      }
      console.error("NodeInfo request failed with status " + response.status);
      return null;
    } catch (error) {
      console.error("NodeInfo request failed with error: " + error);
      return null;
    }
  };
};

// src/intents.ts
var Intents = class {
  // getIntentsMap retrieves the available Activity Intents templates for the provided data
  static getIntentsMap = async (server, webfingerResult) => {
    var found = false;
    var result = {
      announce: "",
      create: "",
      follow: "",
      like: "",
      object: ""
    };
    const links = webfingerResult.links || [];
    for (const link of links) {
      var relation = link.rel || "";
      var template = link.template || link.href || "";
      switch (relation.toLowerCase()) {
        case "https://w3id.org/fep/3b86/announce":
          result.announce = template;
          found = true;
          continue;
        case "https://w3id.org/fep/3b86/create":
          result.create = template;
          found = true;
          continue;
        case "https://w3id.org/fep/3b86/follow":
          result.follow = template;
          found = true;
          continue;
        case "https://w3id.org/fep/3b86/like":
          result.like = template;
          found = true;
          continue;
        case "https://w3id.org/fep/3b86/object":
          result.object = template;
          found = true;
          continue;
        case "http://ostatus.org/schema/1.0/subscribe":
        case "https://ostatus.org/schema/1.0/subscribe":
          if (result.follow == "") {
            result.follow = template.replaceAll("{uri}", "{object}");
          }
          continue;
      }
    }
    if (found) {
      if (result.follow == "") {
        result.follow = result.object;
      }
      if (result.like == "") {
        result.like = result.object;
      }
      if (result.announce == "") {
        result.announce = result.object;
      }
      return result;
    }
    const softwareName = await NodeInfo.getSoftwareName(server);
    switch (softwareName.toLowerCase()) {
      case "diaspora":
        result.create = server + "/bookmarklet?title={name}&notes={content}&url={inReplyTo}";
        break;
      case "friendica":
        result.create = server + "/compose?title={name}&body={content}";
        break;
      case "glitchcafe":
        result.create = server + "/share?text={content}";
        break;
      case "gnusocial":
        result.create = server + "/notice/new?status_textarea={content}";
        break;
        result.create = server + "/share?text={content}";
        break;
      case "hubzilla":
        result.create = server + "/rpost?title={name}&body={content}";
        break;
      case "mastodon":
      case "hometown":
        result.create = server + "/share?text={content}";
        result.object = server + "/authorize_interaction?uri={object}";
        break;
      case "misskey":
      case "calckey":
      case "fedibird":
      case "firefish":
      case "foundkey":
      case "meisskey":
        result.create = server + "/share?text={content}";
        break;
      case "microdotblog":
        result.create = server + "/post?text=[{name}]({inReplyTo})%0A%0A{content}";
        break;
    }
    return result;
  };
};

// src/as/utils.ts
function toString(value) {
  if (value == void 0) {
    return "";
  }
  switch (typeof value) {
    //
    case "bigint":
      return value.toString();
    case "boolean":
      return value ? "true" : "false";
    case "number":
      return value.toString();
    case "object":
      if (Array.isArray(value)) {
        if (value.length == 0) {
          return "";
        }
        return toString(value[0]);
      }
      if (value instanceof Object) {
        if (typeof value.id === "string") {
          return value.id;
        }
        if (typeof value.href === "string") {
          return value.href;
        }
        if (typeof value.url === "string") {
          return value.url;
        }
        return "";
      }
    case "string":
      return value;
    case "symbol":
      return value.toString();
  }
  return "";
}

// src/as/vocab.ts
var ContextActivityStreams = "https://www.w3.org/ns/activitystreams";

// src/as/object.ts
var Object2 = class {
  #value;
  constructor(value) {
    if (value != void 0) {
      this.#value = value;
    } else {
      this.#value = {};
    }
    if (this.#value["@context"] == void 0) {
      this.#value["@context"] = ContextActivityStreams;
    }
  }
  ///////////////////////////////////
  // Conversion methods
  // fromURL retrieves a JSON document from the specified URL and parses it into the JSONLD struct
  fromURL = async (url, options = {}) => {
    options["headers"] = {
      Accept: "application/activity+json"
    };
    const response = await fetch(url, options);
    if (!response.ok) {
      throw new Error(`Unable to fetch ${url}: ${response.status} ${response.statusText}`);
    }
    const body = await response.text();
    this.fromJSON(body);
    return this;
  };
  // fromJSON parses a JSON string into the JSONLD struct
  fromJSON = (json) => {
    this.#value = JSON.parse(json);
    return this;
  };
  // toObject returns the raw JSON object represented by this JSONLD struct
  toObject = () => {
    return this.#value;
  };
  // toJSON returns a JSON string representation of the JSONLD struct
  toJSON = () => {
    return JSON.stringify(this.#value);
  };
  ///////////////////////////////////
  // Setters
  // set sets a property on the JSONLD struct with the given name and value
  set = (name, value) => {
    this.#value[name] = value;
  };
  ///////////////////////////////////
  // Property conversion methods
  get(namespace, property) {
    var result = this.#value[property];
    if (result != void 0) {
      return result;
    }
    result = this.#value[namespace + ":" + property];
    if (result != void 0) {
      return result;
    }
    switch (namespace) {
      case "as":
        return this.#value["https://www.w3.org/ns/activitystreams#" + property];
      case "emissary":
        return this.#value["https://emissary.dev/ns#" + property];
      case "mls":
        return this.#value["https://purl.archive.org/socialweb/mls#" + property];
      case "sse":
        return this.#value["https://purl.archive.org/socialweb/sse#" + property];
    }
    return void 0;
  }
  getString = (namespace, property) => {
    return toString(this.get(namespace, property));
  };
  getInteger = (namespace, property) => {
    const result = this.get(namespace, property);
    if (result == void 0) {
      return 0;
    }
    switch (typeof result) {
      case "number":
        return Math.floor(result);
      case "string":
        const parsed = parseInt(result);
        if (!isNaN(parsed)) {
          return parsed;
        }
    }
    return 0;
  };
  getArray = (namespace, property) => {
    const result = this.get(namespace, property);
    if (result == void 0) {
      return [];
    }
    if (Array.isArray(result)) {
      return result;
    }
    return [result];
  };
  ///////////////////////////////////
  // Properties
  type = () => {
    return this.getString("as", "type");
  };
  id = () => {
    return this.getString("as", "id");
  };
};

// src/as/actor.ts
var Actor = class extends Object2 {
  //
  ///////////////////////////////////
  // Property accessors
  // icon returns the value of the "icon" property
  icon = () => {
    return this.getString("as", "icon");
  };
  // id returns the value of the "id" property
  id = () => {
    return this.getString("as", "id");
  };
  // name returns the value of the "name" property
  name = () => {
    return this.getString("as", "name");
  };
  outbox = () => {
    return this.getString("as", "outbox");
  };
  preferredUsername = () => {
    return this.getString("as", "preferredUsername");
  };
  summary = () => {
    return this.getString("as", "summary");
  };
  type = () => {
    return this.getString("as", "type");
  };
  ///////////////////////////////////
  // MLS-specific properties
  mlsMessages = () => {
    return this.getString("mls", "messages");
  };
  mlsKeyPackages = () => {
    return this.getString("mls", "keyPackages");
  };
  ///////////////////////////////////
  // Emissary-specific properties
  // emissaryMessages returns the URL for the Emissary-specific messages collection
  // that returns BOTH encrypted and unencrypted messages. This is preferred over mls:messages because it allows the client to receive direct messages that are not encrypted with MLS.
  emissaryMessages = () => {
    return this.getString("emissary", "messages");
  };
  // messages returns the URL for the preferred messages collection,
  // which may be either the Emissary-specific collection (if supported) or
  // the standard mls:messages collection (if Emissary-specific collection is not supported).
  // The boolean return value indicates whether the returned URL is for the
  // Emissary-specific collection (true) or the standard mls:messages collection (false).
  messages = () => {
    const emissaryMessages = this.emissaryMessages();
    if (emissaryMessages != "") {
      return { url: emissaryMessages, plaintext: true };
    }
    const mlsMessages = this.mlsMessages();
    if (mlsMessages != "") {
      return { url: mlsMessages, plaintext: false };
    }
    return { url: "", plaintext: false };
  };
};

// src/camper.ts
var Camper = {
  // render redraws the UX based on the current account list in localStorage
  render: () => {
    const accounts = Camper.getSavedAccounts();
    const loadingIndicators = Array.from(document.getElementsByClassName("camper-loading"));
    loadingIndicators.forEach((element) => element.hidden = true);
    const addAccountButtons = Array.from(document.getElementsByClassName("camper-add-account"));
    addAccountButtons.forEach((element) => {
      const maxAccounts = parseInt(element.getAttribute("max-accounts") || element.getAttribute("data-max-accounts") || "3");
      hideElement(element, accounts.length >= maxAccounts);
      element.blur();
    });
    const addFirstAccountButtons = Array.from(document.getElementsByClassName("camper-add-first-account"));
    addFirstAccountButtons.forEach((element) => {
      hideElement(element, accounts.length != 0);
      element.blur();
    });
    const hasAccountsShow = Array.from(document.getElementsByClassName("camper-show-if-has-accounts"));
    hasAccountsShow.forEach((element) => {
      hideElement(element, accounts.length == 0);
      element.blur();
    });
    const hasAccountsHide = Array.from(document.getElementsByClassName("camper-hide-if-has-accounts"));
    hasAccountsHide.forEach((element) => {
      hideElement(element, accounts.length != 0);
      element.blur();
    });
    const addAnotherAccountButtons = Array.from(document.getElementsByClassName("camper-add-another-account"));
    addAnotherAccountButtons.forEach((element) => {
      hideElement(element, accounts.length == 0);
      element.blur();
    });
    const removeAccountButtons = Array.from(document.getElementsByClassName("camper-remove-accounts"));
    removeAccountButtons.forEach((element) => {
      hideElement(element, accounts.length == 0);
    });
    const likeButtons = Array.from(document.getElementsByClassName("camper-btn-like"));
    likeButtons.forEach((element) => {
      element.disabled = !accounts.some((account) => account.intents.like != "");
    });
    const shareButtons = Array.from(document.getElementsByClassName("camper-btn-share"));
    shareButtons.forEach((element) => {
      element.disabled = !accounts.some((account) => account.intents.create != "");
    });
    const announceButtons = Array.from(document.getElementsByClassName("camper-btn-announce"));
    announceButtons.forEach((element) => {
      element.disabled = !accounts.some((account) => account.intents.announce != "");
    });
    const replyButtons = Array.from(document.getElementsByClassName("camper-btn-reply"));
    replyButtons.forEach((element) => {
      element.disabled = !accounts.some((account) => account.intents.create != "");
    });
    const accountNameElements = Array.from(document.getElementsByClassName("camper-account-name"));
    accountNameElements.forEach((element) => {
      const account = accounts[0];
      if (account != void 0) {
        element.innerText = account.name;
      }
    });
    const accountImageElements = Array.from(document.getElementsByClassName("camper-account-image"));
    accountImageElements.forEach((element) => {
      const account = accounts[0];
      if (account != void 0) {
        element.src = account.iconUrl;
        element.hidden = false;
      } else {
        element.src = "";
        element.hidden = true;
      }
    });
    const accountForms = Array.from(document.querySelectorAll("form.camper-form"));
    accountForms.forEach((form) => {
      form.onsubmit = (event) => {
        event.preventDefault();
        event.cancelBubble = true;
        const fediverseHandle = form.elements.namedItem("username");
        Camper.addAccount(fediverseHandle.value);
      };
    });
    const accountLists = Array.from(document.getElementsByClassName("camper-accounts"));
    accountLists.forEach((element) => {
      if (accounts.length == 0) {
        element.innerHTML = "";
        element.hidden = true;
        return;
      }
      const maxAccountsString = element.getAttribute("max-accounts") || element.getAttribute("data-max-accounts") || "3";
      const maxAccounts = parseInt(maxAccountsString);
      const accountListHTML = accounts.slice(0, maxAccounts).map((account) => `
				<div id="camper-account-${account.id}" class="camper-account" onclick="Camper.doIntent(this, '${account.username}')">
					<img src="${account.iconUrl}" class="camper-account-icon">
					<div class="camper-account-info">
						<div class="camper-account-name">${account.name}</div>
						<div class="camper-account-username">${account.username}</div>
					</div>
					<button class="camper-account-remove-button" onclick="Camper.removeAccount('${account.username}')">Remove</button>
				</div>
			`).join("");
      element.innerHTML = accountListHTML;
      element.hidden = false;
    });
  },
  // addAccount adds a new account to the list and redraws the UX
  addAccount: async (username) => {
    const loadingIndicators = Array.from(document.getElementsByClassName("camper-loading"));
    loadingIndicators.forEach((element) => element.hidden = false);
    const webfingerResult = await WebFinger.getMetadata(username);
    if (webfingerResult == null) {
      Camper.render();
      alert("Unable to look up the account you entered.");
      return;
    }
    var accounts = Camper.getSavedAccounts();
    if (accounts.some((account) => account.username.toLowerCase() == username.toLowerCase())) {
      Camper.render();
      return;
    }
    const actorId = WebFinger.getActivityPubId(webfingerResult);
    if (actorId == "") {
      Camper.render();
      alert("Unable to retrieve the profile for the account you entered.");
      return;
    }
    const activityPubActor = await new Actor().fromURL(actorId);
    accounts.push({
      id: actorId,
      username,
      name: activityPubActor.name(),
      iconUrl: activityPubActor.icon(),
      intents: await Intents.getIntentsMap(actorId, webfingerResult)
    });
    localStorage.setItem("camper", JSON.stringify(accounts));
    const shareButtons = Array.from(document.getElementsByClassName("camper-input"));
    shareButtons.forEach((element) => {
      element.value = "";
    });
    Camper.render();
    const newElement = document.getElementById("camper-account-" + actorId);
    if (newElement != null) {
      newElement.click();
    }
  },
  // removeAccount removes an account from the list and redraws the UX
  removeAccount: (username) => {
    window.event.stopPropagation();
    window.event.preventDefault();
    if (!confirm("Remove this account from this device?")) {
      return;
    }
    var accounts = Camper.getSavedAccounts();
    accounts = accounts.filter((account) => account.username.toLowerCase() != username.toLowerCase());
    localStorage.setItem("camper", JSON.stringify(accounts));
    Camper.render();
  },
  // hasSavedAccounts returns TRUE if there is one or more accounts saved in localStorage
  hasSavedAccounts: () => {
    const accounts = Camper.getSavedAccounts();
    return accounts.length > 0;
  },
  // getSavedAccounts retrieves the list of accounts from localStorage
  getSavedAccounts: () => {
    const accountsString = localStorage.getItem("camper");
    if (accountsString == null) {
      return [];
    }
    return JSON.parse(accountsString) || [];
  },
  // doIntent executes the Activity Intent for a selected account (using the data elements in that node)
  doIntent: (element, username = "") => {
    const parent = element.parentElement;
    if (parent.getAttribute("data-intent") == null) {
      parent.setAttribute("data-intent", "follow");
    }
    if (parent.getAttribute("data-on-success") == null) {
      parent.setAttribute("data-on-success", "(close)");
    }
    if (parent.getAttribute("data-on-cancel") == null) {
      parent.setAttribute("data-on-cancel", "(close)");
    }
    const intentName = parent.getAttribute("data-intent");
    if (intentName == null) {
      console.error("Unable to determine intent for clicked element. Please ensure the element has a 'data-camper-intent' attribute.");
      return;
    }
    const accounts = Camper.getSavedAccounts();
    if (accounts.length == 0) {
      alert("No accounts configured. Please add an account to continue.");
      return;
    }
    let account = accounts.find((account2) => account2.username.toLowerCase() == username.toLowerCase());
    if (account == void 0) {
      account = accounts[0];
    }
    var intentTemplate = account.intents[intentName];
    const matches = intentTemplate.match(/\{[^}]+\}/g) || [];
    const placeholders = matches.map((placeholder) => placeholder.slice(1, -1));
    console.log("Found intent template: " + intentTemplate);
    console.log("Placeholders:", placeholders);
    console.log("Dataset", parent.dataset);
    for (const placeholder of placeholders) {
      var value = parent.getAttribute("data-" + placeholder) || "";
      value = encodeURIComponent(value);
      intentTemplate = intentTemplate.replaceAll("{" + placeholder + "}", value);
    }
    if (intentTemplate == "") {
      alert("The account you selected does not support this action.");
      return;
    }
    console.log("Opening intent URL: " + intentTemplate);
    parent.dispatchEvent(new CustomEvent("camper-hide"));
    window.open(intentTemplate, "_blank", "height=750,width=600");
  }
};
Camper.render();
console.log("CamperJS loaded", Camper);
