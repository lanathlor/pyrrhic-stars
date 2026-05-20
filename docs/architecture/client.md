# Client Architecture

Each class has its own player controller scene in Godot:

- Gunner: FPS camera rig, crosshair HUD, recoil system, projectile/hitscan
- Vanguard: third-person action camera, dodge system, combo input buffer, stamina HUD
- Arcanotechnicien: third-person pulled back, Flux commitment UI, channeling bar, school selector
- Engineer: third-person, deployable placement preview, device management panel, drone-cam toggle
- Blade Dancer: third-person target-lock, configuration state display, 4 dynamic ability buttons
- Tutelaire: third-person, aura radius visualization, quick-swap aura selector, targeted projection crosshair

Shared underlying character node: position, stats, health, Flux, inventory. Only the "view layer" changes per class.
