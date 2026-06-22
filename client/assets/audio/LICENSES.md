# Audio asset licenses

Pyrrhic Stars is open source, so every bundled audio file must be CC0 or otherwise
license-compatible. **Policy: every file is attributed in the log below, even CC0 ones
that don't legally require it.** Files under a license that *requires* attribution
(e.g. CC-BY) must additionally have their credit surfaced in-game (e.g. a credits
screen) before release.

## How the files are wired

The logical sound names live in `client/scripts/autoload/audio_manager.gd` (`SOUNDS`).
Each name maps to a path under `sfx/`. Drop a matching `.ogg` (preferred) or `.wav`
in, and it plays automatically the next time that sound fires - no code change. A
missing file warns once and stays silent, so the game runs fine before assets land.

Expected files:

| Logical name          | File                                          |
| --------------------- | --------------------------------------------- |
| `impact_enemy` (Vanguard hit) | `sfx/combat/impact_enemy.ogg`         |
| `impact_player`       | `sfx/combat/impact_player.ogg`                |
| `heal`                | `sfx/combat/heal.ogg`                          |
| `vanguard_cleave`/`upheaval`/`vortex`/`execution` (attack swings) | `sfx/abilities/vanguard_swing.ogg` (shared) |
| `vanguard_block`      | `sfx/abilities/vanguard_block.ogg`            |
| `vanguard_dodge`      | `sfx/abilities/vanguard_dodge.ogg`            |
| `harmonist_cast`      | `sfx/abilities/harmonist_cast.ogg`            |
| `harmonist_beam`      | `sfx/abilities/harmonist_beam.ogg`            |
| `harmonist_zone`      | `sfx/abilities/harmonist_zone.ogg`            |
| `gust_step`           | `sfx/abilities/gust_step.ogg`                 |
| `gunner_fire`         | `sfx/abilities/gunner_fire.ogg`               |
| `footstep`            | `sfx/movement/footstep.ogg`                   |
| `dungeon_start` (cue) | `sfx/cue/dungeon_start.ogg`                    |
| `arena`               | `sfx/ambiance/arena.ogg`                       |
| `lobby` (music)       | `music/lobby.ogg`                              |
| `hub` (music)         | `music/hub.ogg`                                |
| `ui_click`            | `sfx/ui/ui_click.ogg`                          |
| `ui_confirm`          | `sfx/ui/ui_confirm.ogg`                        |

## Attribution log

CC-BY files REQUIRE the credit to be surfaced in-game (e.g. a credits screen) before
release. The HQ preview was downloaded (login-gated originals were not) and converted
to OGG (Vorbis q5).

| File | Source (URL) | Author | License |
| ---- | ------------ | ------ | ------- |
| `sfx/movement/footstep.ogg` | [freesound 268758](https://freesound.org/people/deleted_user_5093904/sounds/268758/) | deleted_user_5093904 | CC-BY 3.0 |
| `sfx/cue/dungeon_start.ogg` | [freesound 333618](https://freesound.org/people/Cooltron/sounds/333618/) | Cooltron | CC-BY 3.0 |
| `music/hub.ogg` | [freesound 789320](https://freesound.org/people/Matio888/sounds/789320/) | Matio888 | CC-BY 4.0 |
| `music/lobby.ogg` | [freesound 858082](https://freesound.org/people/AlekseyKovalenko/sounds/858082/) | AlekseyKovalenko | CC0 (attributed anyway) |
| `sfx/ambiance/arena.ogg` | [freesound 239939](https://freesound.org/people/GregorQuendel/sounds/239939/) | GregorQuendel | CC0 (attributed anyway) |
| `sfx/ui/ui_click.ogg` | [freesound 736912](https://freesound.org/people/jackosnb/sounds/736912/) | jackosnb | CC0 (attributed anyway) |
| `sfx/ui/ui_confirm.ogg` | [freesound 546974](https://freesound.org/people/finix473/sounds/546974/) | finix473 | CC0 (attributed anyway) |
| `sfx/combat/impact_enemy.ogg` | [100 CC0 SFX](https://opengameart.org/content/100-cc0-sfx) (`slam_04`, processed: low-pass + low boost for a deep thud) | OpenGameArt | CC0 (attributed anyway) |
| `sfx/abilities/vanguard_swing.ogg` | [freesound 194081](https://freesound.org/people/potentjello/sounds/194081/) | potentjello | CC0 (attributed anyway) |
| `sfx/abilities/gunner_fire.ogg` | [freesound 440668](https://freesound.org/people/SeanSecret/sounds/440668/) (trimmed to the ~0.18s shot onset) | SeanSecret | CC0 (attributed anyway) |
| `sfx/combat/impact_player.ogg` | [100 CC0 SFX](https://opengameart.org/content/100-cc0-sfx) (`hit_03`) | OpenGameArt | CC0 (attributed anyway) |
| `sfx/combat/heal.ogg` | [100 CC0 SFX](https://opengameart.org/content/100-cc0-sfx) (`bell_01`, trimmed) | OpenGameArt | CC0 (attributed anyway) |
| `sfx/abilities/vanguard_block.ogg` | [100 CC0 SFX](https://opengameart.org/content/100-cc0-sfx) (`metal_03`) | OpenGameArt | CC0 (attributed anyway) |
| `sfx/abilities/vanguard_dodge.ogg` | [Swishes Sound Pack](https://opengameart.org/content/swishes-sound-pack) (`swish-7`) | OpenGameArt | CC0 (attributed anyway) |
| `sfx/abilities/gust_step.ogg` | [Swishes Sound Pack](https://opengameart.org/content/swishes-sound-pack) (`swish-13`) | OpenGameArt | CC0 (attributed anyway) |
| `sfx/abilities/harmonist_cast.ogg` | [Magic Spell SFX](https://opengameart.org/content/magic-spell-sfx) (`magical_6`) | OpenGameArt | CC0 (attributed anyway) |
| `sfx/abilities/harmonist_beam.ogg` | [Magic Spell SFX](https://opengameart.org/content/magic-spell-sfx) (`magical_4`) | OpenGameArt | CC0 (attributed anyway) |
| `sfx/abilities/harmonist_zone.ogg` | [Magic Spell SFX](https://opengameart.org/content/magic-spell-sfx) (`magical_3`) | OpenGameArt | CC0 (attributed anyway) |
