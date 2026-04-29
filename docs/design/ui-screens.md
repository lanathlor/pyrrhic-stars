# UI Screens & Menus

## Purpose

Apply the HUD graphical language to the rest of the interface:

- pause menu
- title/menu flow
- character selection
- character creation
- hub prompts
- group UI
- invite and error popups

These screens should feel like they belong to the same game as the HUD.
They should not switch into a different visual grammar such as "fantasy RPG window", "default engine widget set", or "full-screen launcher app".

The target remains minimalist MMO utility with restrained ElvUI influence:

- compact
- flat
- dark
- aligned
- low decoration
- readable at speed

## Global Rules

### 1. Menus are lighter than traditional MMO windows

Most MMO menus over-frame everything.
This project should not.

Use:

- dark surfaces
- thin borders
- controlled spacing
- clear hierarchy

Avoid:

- thick panel stacks
- ornamental framing
- giant headers
- floating cards with excessive padding

### 2. Each screen needs one visual center of gravity

Every screen should have one obvious focal block.
Examples:

- pause menu: the action list
- title screen: the play entry point
- character select: the selected character list row
- character creation: the class choice area

Do not give equal visual weight to everything.

### 3. Full-screen overlays should use atmosphere, not boxes

When a screen covers the whole viewport, the background treatment should do part of the work.
Prefer:

- dimmed world
- dark translucent wash
- subtle gradient
- sparse accent light

Do not solve the whole composition with one giant centered rectangle.

### 4. Inputs and buttons share the same grammar as HUD slots

Buttons and inputs should inherit the same shape language:

- flat dark fill
- 1px border
- restrained hover/active state
- no rounded toy-like controls
- no thick bevels

The UI should feel engineered, not bubbly.

## Pause Menu

### Role

The pause menu is a temporary interruption, not a destination.
It must be immediate and quiet.

### Composition

- dark screen wash over gameplay
- short vertical action list
- no giant framed modal if it can be avoided
- title optional, but if present it should be small

### Visual behavior

- current selection/hover gets the strongest contrast
- actions are stacked cleanly with equal width
- quit and return-to-menu should be visually subordinate to resume

### Tone

Functional, not dramatic.
The player should feel one step away from gameplay.

## Title / Main Menu

### Role

The title screen is the first branding moment, but it still should not become loud.
This project is not selling whimsy or spectacle through menu chrome.

### Composition

- one strong title mark
- one short supporting subtitle or environment descriptor
- one clear call to action
- optional account welcome state

### Layout

Keep the interaction block narrow and centered.
Do not spread controls across the screen.

Use:

- centered column
- generous vertical rhythm
- large title
- smaller functional controls below

### Visual language

- title may carry a bit more contrast than the gameplay HUD
- buttons stay restrained
- background should carry mood through atmosphere rather than panels

## Character Selection

### Role

Character select is a roster management screen.
The player is comparing entries and choosing one, not reading lore.

### Main pattern

The character list is the screen.
Rows should be the dominant unit.

### Row design

Each character row should be:

- flat
- horizontally structured
- easy to scan
- easy to highlight

Recommended row contents:

- character name
- class
- optional last-played marker
- optional readiness or progression cue later

### Selection state

Selection should be obvious through:

- accent border
- slightly brighter fill
- maybe a slim edge line

Do not rely on a giant glow or animated frame.

### Empty state

If there are no characters:

- keep the message short
- keep the action obvious
- do not center it inside a large ornamental box

## Character Creation

### Role

Character creation is a decision screen.
It must help the player choose a class quickly and confidently.

### Priority order

1. Class identity
2. Class playstyle
3. Name entry
4. Confirmation

### Class presentation

Class cards should not feel like chunky dashboard widgets.
They should read more like tactical selection tiles.

Each class tile should emphasize:

- class name
- gameplay genre
- short fantasy or combat identity

Keep text brief.
If the class requires explanation, use a side detail area or tooltip rather than filling the card with paragraphs.

### Selected class

The selected tile should gain:

- sharper border
- accent color
- maybe slightly darker or brighter interior

It should not gain a heavier box shadow or oversized panel.

### Name entry

Name entry belongs below the class choice, because it is a confirmation step, not the main decision.

Keep it:

- centered
- compact
- visually subordinate to class selection

### Error presentation

Errors should be direct and local:

- inline red text
- short sentence
- no modal popup for a simple validation failure

## Hub UI

### Role

Hub UI is contextual guidance layered over free movement.
It should stay lighter than combat HUD and much lighter than full menus.

### Elements

- status text
- interaction prompt
- portal prompt
- lift prompt
- optional class switch hint if still needed for development

### Rules

- prompts should appear near the player focus zone or screen center
- persistent hub guidance should stay near edges or top-center
- avoid large opaque backgrounds behind prompts
- use color to signal interactability, not decoration

### Prompt styling

Prompts should look ephemeral:

- text first
- maybe a thin backing strip if readability needs it
- accent color only on the actionable token

## Group UI

### Role

Group UI exists in two different modes:

1. ambient management in the hub
2. live party awareness in the HUD

This document covers the ambient management side.

### Group panel in hub

The hub group panel should not feel like a standalone menu window.
It is a side utility panel.

It should be:

- narrow
- edge-aligned
- low contrast
- primarily text plus one or two actions

### Information priority

1. current group state
2. member list
3. leader
4. leave/create action

### Visual treatment

- thin outline or subtle backing
- stacked member list
- no oversized header
- keep the buttons small and functional

## Invite Popups

### Role

Invite popups are interruptive but low stakes.
They need immediate clarity, not dramatic presentation.

### Requirements

- inviter name is the first thing seen
- accept and decline are visually distinct
- popup should appear close to screen center or upper center
- popup footprint should be compact

### Styling

- small dark block or strip
- thin border
- one accent for accept
- danger or muted color for decline

Avoid the look of a quest dialog or cinematic prompt.

## Error and Status Messages

### Rule

Use the lightest mechanism that can communicate the problem.

Examples:

- inline validation text for naming errors
- short top-center message for transient status
- popup only for actionable decisions

### Tone

System messages should be:

- short
- factual
- non-diegetic
- easy to parse in one glance

## Widget Rules

### Buttons

Buttons should look like compact utility controls.

Use:

- flat dark fill
- thin border
- subtle hover brightening
- stronger active/selected accent

Avoid:

- thick shadows
- glossy gradients
- highly rounded corners
- giant padding

### Text inputs

Text inputs should be visually quieter than buttons.
They are containers for text, not calls to action.

Use:

- dark inner fill
- clear border
- readable cursor contrast
- compact vertical size

### Scroll areas

Scroll containers should feel almost invisible.

- keep rails subtle
- avoid giant scroll backgrounds
- let content rows carry most of the structure

## Color Use By Screen Type

### Menus

Menus may use slightly more atmospheric contrast than the HUD, but still remain restrained.

### Selection flows

Selection screens use accent color to show the chosen item, not to paint every item.

### Errors

Red is reserved for errors, danger, and hostile threat.
Do not spend that signal on decoration.

### Confirmation

Use cool or neutral accents for confirmation and navigation.
Do not turn every primary button into a bright saturated block.

## Spacing Rules

The current repo UI often uses large container margins and large gaps between elements.
That should change.

General target:

- tighten outer margins
- reduce dead vertical space
- keep screen center readable
- let alignment create order instead of empty space

If a panel feels "clean" only because it has a lot of blank area, it is probably too large.

## Implementation Guidance For This Repo

The current non-HUD UI in `client/scenes/main.gd` should be refactored toward a shared style system.

Recommended reusable primitives:

- `UiButton`
- `UiInput`
- `UiRow`
- `UiPrompt`
- `UiOverlay`
- `UiListPanel`
- `UiSelectionTile`

The important point is not component count.
The important point is that pause, menu, character, and group interfaces stop inventing separate looks.

## Practical Test

A non-HUD screen matches the intended language if:

1. It still reads clearly after removing 20-30% of its panel chrome.
2. The primary action is obvious within one second.
3. Selected state is readable without animation.
4. The screen feels related to the combat HUD.
5. It does not look like a generic engine default menu.
