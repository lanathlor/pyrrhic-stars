# Classes

6 classes. Each plays a fundamentally different game genre. The camera, input model, HUD, and core gameplay loop differ per class.

| Class                           | Genre                       | Flux Usage |
| ------------------------------- | --------------------------- | ---------- |
| [Gunner](gunner.md)             | First-Person Shooter        | Minimal    |
| [Vanguard](vanguard.md)         | Souls-like Action Melee     | Minimal    |
| [Arcanotechnicien](arcanotechnicien.md) | Tactical Flux Channeling    | Primary    |
| [Engineer](engineer.md)         | Deployable Management       | Moderate   |
| [Blade Dancer](blade-dancer.md) | Positional Combo Fighter    | Moderate   |
| [Tutelaire](tutelaire.md)       | Aura Positioning and Triage | Heavy      |

## Specialization Design Axes

Every DPS spec is positioned on two axes: **monotarget vs AoE** and **burst vs constant damage**. Tank and healer specs operate outside this framework.

### DPS Spec Matrix

|              | Monotarget                           | AoE                                          |
| ------------ | ------------------------------------ | -------------------------------------------- |
| **Burst**    | Marksman, Dual Blade, Pilot        | Destroyer, Chasseur, Blade              |
| **Constant** | Assault, Shadow, Battlemage          | Architect, Multi Blade, Adjudicator     |

### Full Spec Breakdown

| Class            | Spec         | Role   | Target | Damage   | Identity                                       |
| ---------------- | ------------ | ------ | ------ | -------- | ---------------------------------------------- |
| **Gunner**       | Assault      | DPS    | Mono   | Constant | High fire rate, aggressive repositioning       |
|                  | Marksman     | DPS    | Mono   | Burst    | Slow, deliberate, perfect shots                |
|                  | Chasseur     | DPS    | AoE    | Burst    | Grenades, EMP, area denial                     |
| **Vanguard**     | Blade        | DPS    | AoE    | Burst    | Blade swirl, ground slam, commit-to-cleave     |
|                  | Shield       | Tank   | —      | —        | Directional block, absorbs for allies          |
|                  | Shadow       | DPS    | Mono   | Constant | Counters, flanking, sustained stealth pressure |
| **Arcanotechnicien** | Destroyer    | DPS    | AoE    | Burst    | Massive abilities, long channels, huge payoff  |
|                  | Battlemage   | DPS    | Mono   | Constant | Melee hybrid, weaving strikes with abilities   |
|                  | Harmonist    | Healer | —      | —        | Channeled healing zones and beams              |
| **Engineer**     | Architect    | DPS    | AoE    | Constant | Turrets, barricades, zone control              |
|                  | Pilot        | DPS    | Mono   | Burst    | Single powerful drone, focused fire            |
|                  | Saboteur     | DPS    | AoE    | Burst    | EMP, overload, disruption fields               |
| **Blade Dancer** | Dual Blade  | DPS    | Mono   | Burst    | 2 blades, separate GCDs, piano burst combos    |
|                  | Multi Blade  | DPS    | AoE    | Constant | 4-6 blades, Scatter/Fan sustained multi-target |
| **Tutelaire**    | Guardian     | Tank   | —      | —        | Damage reduction auras, solid light barriers   |
|                  | Adjudicator  | DPS    | AoE    | Constant | Retribution aura, ticking damage               |
|                  | Luminary     | Healer | —      | —        | Healing auras, beacons, channeled restoration  |

### Known Imbalances

-   **Burst/mono is overcrowded** (4 specs) vs **burst/AoE** (2 specs). Mitigated by each spec playing a completely different genre.
-   **Chasseur** could shift to mono/constant (surgical anti-channeler) to better match its Rainbow Six identity and differentiate from Assault.
-   **Saboteur vs Destroyer** both occupy AoE/burst. Saboteur could shift to AoE/constant (persistent EMP fields, DoT hacking) to differentiate.
