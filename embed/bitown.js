/**
 * bitown embed loader (vanilla JS, zero deps).
 *
 * Usage:
 *   <div data-bitown-city="testcity" data-bitown-width="200" data-bitown-height="240"></div>
 *   <script src="https://bitown.dev/embed/bitown.js" async></script>
 *
 * Optional attributes on the host element or script tag:
 *   data-bitown-api="https://bitown.dev"   — API origin (default: script origin)
 *   data-bitown-width / data-bitown-height — iframe size (default 200×240)
 */
(function () {
  "use strict";

  function scriptOrigin() {
    var scripts = document.getElementsByTagName("script");
    for (var i = scripts.length - 1; i >= 0; i--) {
      var src = scripts[i].src || "";
      if (src.indexOf("bitown.js") !== -1) {
        try {
          return new URL(src).origin;
        } catch (e) {
          /* ignore */
        }
      }
    }
    return "";
  }

  function attr(el, name, fallback) {
    var v = el.getAttribute(name);
    return v != null && v !== "" ? v : fallback;
  }

  function isAllowedApi(api, scriptBase) {
    try {
      var origin = new URL(api).origin;
      return (
        origin === scriptBase ||
        origin.indexOf("://localhost") !== -1 ||
        origin.indexOf("://127.0.0.1") !== -1
      );
    } catch (e) {
      return false;
    }
  }

  function mount(host) {
    if (host.getAttribute("data-bitown-mounted") === "1") return;
    var slug = attr(host, "data-bitown-city", "");
    if (!slug) return;

    var scriptBase = (scriptOrigin() || window.location.origin).replace(/\/$/, "");
    var requested = attr(host, "data-bitown-api", "").replace(/\/$/, "");
    var api = scriptBase;
    if (requested && isAllowedApi(requested, scriptBase)) {
      api = requested;
    }
    var width = attr(host, "data-bitown-width", "200");
    var height = attr(host, "data-bitown-height", "240");
    // Widget HTML always loads from the script origin (trusted host).
    var widget = scriptBase + "/embed/widget.html?slug=" + encodeURIComponent(slug) +
      "&api=" + encodeURIComponent(api);

    var iframe = document.createElement("iframe");
    iframe.src = widget;
    iframe.title = "bitown city " + slug;
    iframe.width = String(width);
    iframe.height = String(height);
    iframe.loading = "lazy";
    iframe.setAttribute("scrolling", "no");
    iframe.style.border = "0";
    iframe.style.maxWidth = "100%";
    iframe.style.overflow = "hidden";
    iframe.style.display = "block";

    host.appendChild(iframe);
    host.setAttribute("data-bitown-mounted", "1");
  }

  function boot() {
    var nodes = document.querySelectorAll("[data-bitown-city]");
    for (var i = 0; i < nodes.length; i++) mount(nodes[i]);
  }

  if (document.readyState === "loading") {
    document.addEventListener("DOMContentLoaded", boot);
  } else {
    boot();
  }
})();
