# Windows via WSL — research (2026-11-01, first pass; web search only, nothing tested)

**Question (Christopher):** WSL is now open source. Could Windows be served by a one-click WSL install of our slim Alpine Linux image instead of a separate Windows build, for the desktop app and the server?

## Findings

- **One-file install exists.** A WSL distro can ship as a single `.wsl` file: a root filesystem tar plus WSL config, renamed `.wsl`. The user double-clicks it in Explorer or runs `wsl --install --from-file <file>`. `/etc/wsl-distribution.conf` must set `oobe.defaultName` for the double-click install to work. Only Microsoft Store listing needs Microsoft approval; a file we hand out does not.
- **GUI apps work through WSLg** (built into WSL 2: Wayland/X server in a hidden system distro). Apps show as normal Windows windows. WSL 2 only. Not a full desktop; Microsoft calls GUI support limited.
- **Alpine:** our server image is already slim Alpine, so the WSL image can be built from the same base (one Linux image to maintain). Alpine packages WebKitGTK (what Wails v2 draws with) on musl. **Unproven:** no report found of Wails/WebKitGTK on Alpine under WSLg. Needs a real Windows test.

## Costs / risks

- First WSL use on a PC needs CPU virtualization enabled, admin rights, and usually a reboot. Not one-click for a non-technical owner.
- WSL runs a lightweight VM; Alpine keeps it small and it releases memory when idle, but it sits on top of Windows.
- No paid code signing (ruled 10-19), so Windows may warn on our installer file.

## Server side: not a fit

WSL stops when idle, does not start at boot without a scheduled-task workaround, sits behind Windows NAT networking (mirrored mode helps, extra setup), and goes down with Windows sleep/update reboots. Servers stay Linux (our hosted laptop, Proxmox helper, CasaOS).

## Recommendation (Claude, 2026-11-01)

1. Windows players: the PWA in the browser (already approved) — nothing to install, no Windows build.
2. Windows players who want the desktop app: optional `.wsl` Alpine package built from the server image.
3. Servers: Linux only, never WSL.

## Next step when picked up

Windows VM (already planned before item 52): install a test `.wsl` Alpine image, launch the desktop app under WSLg, check it draws, measure idle RAM, and time a clean first-install (virtualization, admin, reboot).

## Sources

- https://learn.microsoft.com/windows/wsl/build-custom-distro
- https://hothardware.com/news/wsl-sample-open-source
- https://github.com/Microsoft/WSLg
- https://github.com/MicrosoftDocs/wsl/blob/master/WSL/tutorials/gui-apps.md
- https://awesome.ecosyste.ms/projects/github.com%2Fdeveloperscommunity%2Farchlinux-wsl
