// src/utils.ts
function guessProtocol(server) {
  switch (server) {
    case "localhost":
    case "127.0.0.1":
      return "http://";
  }
  return "https://";
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
export {
  WebFinger
};
