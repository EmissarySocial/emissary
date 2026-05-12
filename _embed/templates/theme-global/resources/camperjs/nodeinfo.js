// src/utils.ts
function guessProtocol(server) {
  switch (server) {
    case "localhost":
    case "127.0.0.1":
      return "http://";
  }
  return "https://";
}

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
export {
  NodeInfo
};
