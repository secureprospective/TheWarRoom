# Desktop glance widget research

**Recommendation:** For the fastest working prototype, build a **Document Picture-in-Picture (PiP) view for the web/PWA**. For a widget integrated into the existing Wails desktop app, investigate **Wails v3 multi-window support**—but treat migration and Linux window-manager behavior as separate work. Neither route guarantees a retractable screen-edge tab on every OS.

## 1. Browser and PWA

**Document PiP** opens arbitrary HTML in a small, always-on-top window—not just video.

- **Support:** Desktop Chrome and Edge support it from version 116. Current desktop Brave is Chromium-based and likely supports it; feature-detect and test the installed version. Firefox and Safari do not support Document PiP, although they have separate video-PiP features. [MDN](https://developer.mozilla.org/en-US/docs/Web/API/Document_Picture-in-Picture_API)
- **Limits:** Opening normally requires a user gesture. Request an initial width and height, but the browser can clamp them. Users can resize; programmatic resizing requires a gesture and is subject to browser limits. Only one Document PiP window is allowed per opener; opening another may replace an existing PiP window. It closes when its opener closes or navigates. [Chrome documentation](https://developer.chrome.com/docs/web-platform/document-picture-in-picture), [specification](https://github.com/WICG/document-picture-in-picture)
- **Tab versus installed PWA:** PiP is a separate window tied to the originating document; installing the PWA does not turn the PWA’s main window into an independently persistent widget. Test the installed experience on target systems.
- **Edge tab:** The user can move the PiP window, but the site cannot set its screen position. A reliable programmed edge-dock or auto-retract tab is therefore **not available** through this API.
- **Production example:** Spotify uses Document PiP and a React portal for its web and desktop miniplayer. [Chrome case study](https://developer.chrome.com/blog/spotify-picture-in-picture)

**Windows 11 Widgets board:** An installed PWA can provide a widget using a manifest, service worker, Adaptive Cards and data. This is a **board widget**, not a free-floating always-on-top window or a normal React page. [Microsoft guide](https://learn.microsoft.com/en-us/microsoft-edge/progressive-web-apps/how-to/widgets)

## 2. Wails desktop

- **v2:** Supports a frameless, resizable, always-on-top **main window**, but does not natively provide the second window and integrated tray needed for this design. A separate companion process or custom native integration is possible, with added lifecycle and data-sharing work. [v2 options](https://v2.wails.io/docs/reference/options/), [Wails FAQ](https://wails.io/faq/)
- **v3:** Supports multiple windows and a system tray, making a second frameless, resizable, always-on-top window practical. Migration from v2 is a port, not just a dependency bump. Pin and test a release; [v3 releases](https://github.com/wailsapp/wails/releases) are available. [Window options](https://wails.io/features/windows/options/), [migration guide](https://docs-preview.wails.io/migration/v2-to-v3/)
- **Docking:** Edge snapping, retracting to a narrow tab, and restoring size would be custom behavior, not something to assume Wails supplies out of the box. Test monitor boundaries, scaling and hot-plugging.
- **OS caveats:** On Linux, always-on-top and positioning depend on the window manager; Wayland compositors may restrict or ignore them. On macOS, floating level alone does not guarantee visibility across Spaces or over fullscreen apps; appropriate collection behaviors and testing are needed. Windows also needs testing against fullscreen apps and OS window-management behavior. [Wails window options](https://v3.wails.io/features/windows/options/)

## 3. Effort and data

**Rough engineering estimates**, assuming an existing authenticated scores API and reusable React components:

| Route | Prototype | Main uncertainty |
|---|---:|---|
| Web/PWA Document PiP | **1–3 days** | Browser/installed-PWA testing and polish |
| Wails v3 second window and tray | **3–7 days after migration** | Migration effort, OS-specific behavior |
| Retractable edge tab | **Additional 3–10+ days** | Platform-specific implementation; Wayland may prevent parity |

For PiP, render the glance component through a React portal and reuse the opener’s existing state and backend subscriptions. For v3, have both windows call the same Go service/cache, with events pushing updates to each window. Neither requires a second scores backend.

**First step:** Prototype the React PiP view in Chrome, then verify live updates, resize, close/reopen, and installed-PWA behavior. Separately test a minimal Wails v3 two-window spike on the actual Linux session before committing to migration.
