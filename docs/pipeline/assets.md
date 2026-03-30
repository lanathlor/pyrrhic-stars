# Asset Pipeline

## Tools

- **Blender**: modular kit pieces, props, materials, export to GLB
- **Free asset packs (CC0)**: Kenney, Quaternius, PolyHaven, Sketchfab (CC0), itch.io packs as base assets
- **Mixamo**: free rigged humanoid characters and animation library
- **Godot 3D editor**: level assembly, lighting, particle systems
- **Godot Terrain3D plugin**: outdoor zones (later phases)

## Workflow

1. Design dungeon layout on paper/markdown (top-down map, encounter placement, flow)
2. Source or model modular kit pieces and props in Blender, export as GLB
3. Build CSG blockout in Godot (grey-box playable layout)
4. Playtest blockout for flow and spacing
5. Replace CSG geometry with actual GLB assets
6. Polish pass in Godot editor: lighting tweaks, fog density, prop nudging
7. Write encounter scripts, enemy AI, boss phases, spawn triggers

## Art Direction

Stylized realism, not photorealism. Strong art direction with simpler geometry. Think Warframe or Destiny's mood, not their polygon count.

Sci-fi hard-surface assets (guns, armor, corridors, tech) work well with modular kits and CC0 packs. The setting leans into what can be built efficiently with free tooling.

Dark environments with atmospheric lighting hide geometric simplicity. Rain, fog, volumetric light, wet surface reflections do 80% of the visual work.
