// src/utils.ts
function guessProtocol(server) {
  switch (server) {
    case "localhost":
    case "127.0.0.1":
      return "http://";
  }
  return "https://";
}
function getPlaceholders(template) {
  const matches = template.match(/\{([^}]+)\}/g) || [];
  return matches.map((placeholder) => placeholder.slice(1, -1));
}
function safeText(value) {
  const parser = new DOMParser();
  const parsed = parser.parseFromString(value, "text/html");
  return parsed.body.textContent || "";
}
function safeURL(value) {
  if (value == "") {
    return "";
  }
  let parsed;
  try {
    parsed = new URL(value, document.baseURI);
  } catch {
    return "";
  }
  switch (parsed.protocol) {
    case "http:":
    case "https:":
      return parsed.href;
  }
  return "";
}
function safeAttr(value) {
  return value.replaceAll("&", "&amp;").replaceAll("<", "&lt;").replaceAll(">", "&gt;").replaceAll('"', "&quot;").replaceAll("'", "&#39;");
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
export {
  getPlaceholders,
  guessProtocol,
  hideElement,
  safeAttr,
  safeText,
  safeURL
};
