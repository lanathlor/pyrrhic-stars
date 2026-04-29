# HUD Graphical Language

## Purpose

Define the visual language of the in-game HUD so every widget follows the same rules.
The target is a minimalist MMO HUD influenced by ElvUI: compact, readable, low-noise, and mechanically dense without looking ornamental.

This is not a 1:1 copy of ElvUI.
The goal is to borrow its discipline:

- very low decoration
- strong alignment
- thin borders
- dark neutral surfaces
- restrained color usage
- information first, chrome second

## Core Principles

### 1. Function over decoration

Every visible line, block, and label must justify itself.
If a frame exists only to suggest "this is a UI box", remove it.
Containers should appear only when they improve readability or grouping.

### 2. Density without clutter

The HUD should carry a lot of combat information, but it must remain visually quiet.
This means:

- short labels
- compact bars
- limited icon framing
- no redundant titles like `MINIMAP`, `ACTION BAR`, or `STANCE`
- no repeated labels when the object is already self-evident

### 3. Neutral base, selective accent

The default HUD should be mostly dark grey-blue neutrals.
Accent colors are reserved for state:

- class identity
- danger
- buffs/procs
- resource type
- interactable focus

Accent should not become background.
Most of the screen should remain neutral.

### 4. Shape consistency

The base grammar is rectilinear:

- straight edges
- narrow borders
- flat fills
- no heavy bevels
- no ornamental corners

Circular shapes are allowed only when the mechanic itself is circular:

- minimap
- radial cooldown or lock-on feedback
- target markers

### 5. Screen humility

The HUD must never dominate the 3D scene.
It should feel attached to the screen edges and combat center, not floating as large interface windows.

## Visual Vocabulary

### Surfaces

Use three surface levels only:

1. `Base`
   Very dark background for bars and slot interiors.
2. `Raised`
   Slightly brighter neutral fill for a grouped control if grouping is required.
3. `Overlay`
   Temporary feedback such as damage flash, proc highlight, or cooldown cover.

Surface contrast should stay low.
The player should distinguish states by layout and fill behavior, not by loud panels.

### Borders

Borders are structural, not decorative.

- 1px most of the time
- muted grey-blue
- used to define edges of bars and slots
- never doubled unless the inner line conveys a real state

If a border can be removed without hurting readability, remove it.

### Color Roles

- `Health`: green at stable state, red only when critical or hostile
- `Primary resource`: gold, cyan, or another class-specific resource color
- `Danger`: red
- `Friendly`: green
- `Neutral chrome`: blue-grey
- `Class accent`: one controlled accent per class

Color should encode meaning, not decoration.
For example, the Blade Dancer config color is meaningful because it communicates state.

### Typography

Typography should be sparse and subordinate to the bars.

- prefer short text blocks
- avoid all-caps unless the term is very short
- numeric values matter more than labels
- labels should sit inside or directly adjacent to the element they describe
- no header text for obvious elements

The most important text is:

- current values
- mode/config names
- target names
- combat timing or cooldown numbers

## Layout Rules

### Bottom center

Bottom center is the primary personal information zone.
It contains:

- player health/resource
- action bar
- class-specific mode display

These elements should read as one cluster, with narrow vertical spacing.
They should not be wrapped in a giant parent frame.

### Top center

Top center is reserved for encounter authority:

- boss name
- boss health
- phase information

This region should be flatter than the bottom cluster.
The bar itself is the element.
Any outer shell should be minimal or absent.

### Left side

Left side is for party information.
Party frames must be compact and stack cleanly.
They should favor:

- name
- health
- maybe one compact secondary signal

They should not look like separate windows.

### Right side

Right side holds secondary analytical information such as damage meter or combat log.
This content is important but not urgent, so it must stay visually quiet.

### Top right

The minimap is an exception because its circular geometry is functional.
The circle itself is enough.
Avoid boxing the minimap inside another rectangle unless required by readability.

## Component Rules

### Bars

Bars are the main unit of the UI language.

Bars should be:

- thin
- long enough to read at a glance
- flat-filled
- bordered lightly

Use stacked bars instead of boxed meters whenever possible.

### Slots

Action slots are allowed more structure than bars because they contain multiple signals:

- keybind
- icon or spell identity
- cooldown state
- proc/active highlight

Even then, slots should remain compact.
The action bar should not sit inside a large dock unless the dock communicates a real grouping need.

### Mode displays

Mode displays like Blade Dancer config should be mostly text plus a very small secondary marker system.
For example:

- mode name
- a row of pips
- color-coded state

That is enough.
It does not need a banner.

### Tooltips

Tooltips may use more containment than the always-visible HUD because they are temporary and informational.
Still:

- keep them small
- avoid thick framing
- put the spell name first
- keep description density controlled

## Motion and Feedback

Animation should be purposeful and short.

- cooldown sweep
- proc pulse
- hit marker fade
- damage flash
- lock-on feedback

Avoid ambient animation in passive HUD regions.
Idle motion makes a minimalist UI feel noisy.

## What To Avoid

- oversized parent frames
- nested frame-inside-frame chrome
- decorative headers on obvious elements
- large padding around small content
- class branding everywhere
- gradients that attract more attention than the mechanic
- bright color used as default background
- MMO "window" look

## Practical Test

A HUD element matches this language if:

1. Removing 30% of its chrome does not improve it.
2. The state can be understood in under a second.
3. The element still feels light over gameplay footage.
4. Accent color is doing semantic work, not cosmetic work.
5. The element can sit next to another HUD element without both feeling like separate panels.

## Current Direction In This Repo

The current HUD pass should continue moving toward:

- thinner bars
- fewer outer shells
- tighter spacing
- less repeated labeling
- clearer state colors

If a future pass has to choose between "looks finished" and "looks light", prefer "looks light".
