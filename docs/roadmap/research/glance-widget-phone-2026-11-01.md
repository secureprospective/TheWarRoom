# Phone glance widget research

**Recommendation:** On Android, prototype a user-started floating overlay if the edge-tab experience is essential; pair it with a Live Update notification. On iPhone, use a Live Activity for the reliable, Apple-supported experience. A true custom edge tab over other apps is not generally available on iOS.

## 1. Floating over other apps

### Android

- **Custom overlay:** `SYSTEM_ALERT_WINDOW` can draw over other apps, so a draggable, collapsible edge tab is technically possible. The user must explicitly grant “display over other apps” access in Settings. Android says very few apps should use this permission; explain the benefit and offer an off switch. [Android permission reference](https://developer.android.com/reference/android/Manifest.permission)
- **Bubbles:** Android 10 introduced them, but on Android 11+ they are primarily for eligible **conversations**, with notification and shortcut requirements. A standalone scoreboard is not a sound fit. Users also control whether bubbles appear. [Bubbles documentation](https://developer.android.com/develop/ui/compose/notifications/bubbles)
- **Picture-in-picture:** Android supports non-video PiP, including ongoing tasks such as navigation. A score view is technically plausible, but PiP has system-controlled sizing and interaction—not a freely designed edge tab. Google’s guidance includes ticker-type use; this is not a guarantee of Play approval. [PiP guidance](https://developer.android.com/distribute/aep/aep-req-picture-in-picture)
- **Battery and policy:** Persistent background work is constrained. A foreground service has a visible notification and must meet its declared purpose and Play requirements; an overlay is not a general exemption. For apps targeting Android 15+, the overlay-related background-service-start exemption requires an **already visible** overlay. Test on actual devices: manufacturers may impose additional battery restrictions. [Service changes](https://developer.android.com/develop/background-work/services/fgs/changes) · [Play policy](https://support.google.com/googleplay/android-developer/answer/18265346)

### iOS

Apps cannot generally place arbitrary interactive windows or edge tabs over other apps. **PiP is the exception to investigate, not a guaranteed workaround:** `AVSampleBufferDisplayLayer` can feed custom-rendered frames to PiP, but Apple documents it as a video presentation mechanism, not a general floating UI. A score-only implementation carries review risk; I found no explicit blanket ban or reliable rejection rate. PiP interaction and placement remain system-controlled. [Apple PiP documentation](https://developer.apple.com/documentation/avkit/adopting-picture-in-picture-in-a-custom-player)

Non-video examples exist: [Floating Notepad](https://apps.apple.com/us/app/floating-notepad-overlay/id1598380826) advertises a floating notes overlay. That demonstrates an available app, **not** approval precedent for our scoreboard or proof of its implementation.

## 2. Sanctioned glanceable options

| Platform | Option | What it gives us |
|---|---|---|
| iOS | **Live Activity** | Matchup score on Lock Screen and, on supported iPhones, Dynamic Island. The Island offers compact and expanded system layouts, not arbitrary resizing. Updates can arrive through APNs. An activity runs up to **8 hours**, then can remain on the Lock Screen up to **4 more**. Activity data has a **4 KB limit**; update delivery is budgeted and can be throttled. Frequent updates require configuration and user permission. [Apple](https://developer.apple.com/documentation/ActivityKit/displaying-live-data-with-live-activities) · [Push updates](https://developer.apple.com/documentation/ActivityKit/starting-and-updating-live-activities-with-activitykit-push-notifications) |
| Android | **Live Updates, Android 16+** | A promoted ongoing notification can show a user-selected game and surface prominently, subject to eligibility, user settings and system decisions. Provide an unpin action. Earlier Android versions can use ordinary ongoing notifications. [Google](https://developer.android.com/develop/ui/views/notifications/live-update) |
| Both | **Home-screen widgets** | Native WidgetKit or Jetpack Glance widgets show scores outside the app, **not over other apps**. Both support interactions; iOS offers system-defined sizes, while Android widgets can be user-resizable. Refresh is power-managed—not a guaranteed live feed. Android’s periodic widget update setting has a 30-minute minimum. [Apple](https://developer.apple.com/documentation/widgetkit/adding-interactivity-to-widgets-and-live-activities) · [Android](https://developer.android.com/develop/ui/views/appwidgets/advanced) |

**Precedent:** [Yahoo Fantasy](https://help.yahoo.com/kb/SLN36451.html) documents Live Activities for head-to-head matchups. I could not verify current fantasy-matchup Live Activity support for ESPN Fantasy or Sleeper; their absence from the sources I found is not proof they lack it.

## 3. PWA versus native shell

| Route | Reach | Rough effort* |
|---|---|---|
| **PWA** | Normal web UI and permission-based web notifications. On iOS, Web Push requires an installed Home Screen web app and iOS 16.4+. No direct access to the native overlay, Live Activity, Live Update or native widget APIs above. [Apple Web Push](https://developer.apple.com/documentation/usernotifications/sending-web-push-notifications-in-web-apps-and-browsers) | Days for basic notifications, excluding backend integration. |
| **Capacitor, React Native or native app** | Can reach these OS features through platform-specific native code/plugins. A shell does **not** turn the existing React screen into a Live Activity or widget automatically; those need native presentation and push integration. | Roughly **2–4 weeks per platform** for a production-minded first glance feature; a custom Android overlay adds testing and policy work. |

\*Planning estimates, not quotes: depend on existing score feeds, push infrastructure, authentication, device coverage and store review.

**First step:** Build an iOS Live Activity proof of concept and an Android 16 Live Update using one real matchup feed. Separately prototype the Android edge tab on two devices before committing to it. For iOS, only explore PiP after deciding whether a review-risk experiment is worth maintaining.
