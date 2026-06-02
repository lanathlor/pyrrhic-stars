---
title: 'Hello, Pyrrhic Stars.'
description: 'I am building a co-op action game where every class plays a different genre. The whole project is open source. Here is why, and what to expect from this devlog.'
pubDate: 2026-06-02
---

Pyrrhic Stars is a co-op action game where every class plays a different genre. One dungeon, one boss, five players, but the way each of you plays the game depends entirely on the class you chose.

If you are reading this, you are probably one of two people. Either you followed a link from somewhere I posted the announcement, or you typed the URL in by accident. Either way, welcome.

## Why I am building this

I have been playing co-op games for twenty years, and they all run on the same template. Three roles, one of each. Healer, tank, DPS, repeat. The roles are different skins over the same job: stand in the right place, push the right buttons, do not die. The class you picked decides which buttons you push. It almost never decides how the game _feels_.

I want to break that.

In Pyrrhic Stars, the Gunner holds the camera at eye level. The Vanguard orbits the boss in third person, looking for an opening, ready to dodge or parry. The Blade Dancer target-locks and works a state machine, swapping configurations as her cooldowns line up. The Arcanotechnicien stops moving, lines up a Flux channel, and _commits_ to an ability for three full seconds while the boss is locked onto her. Four people, one fight, four completely different games happening at the same time. None of them are more important than the others. None of them replace the others. The fight is not winnable if any of them are missing.

That is the game. That is what I am building.

## The code is open source

The whole project lives on a public repository: client, server, design docs, art sources, and the docs you would need to understand any of the above. The link is in the badge at the top of this page.

I do not draw a line between the devlog and the code. The devlog is a tour of commits. If something I describe in a post does not match what is in the repository, the repository is the truth and I have not done my job this week. Every concrete technical claim in this post, from the four classes to the boss patterns to the Flux system to the server tick rate, is something you can open in your editor and read for yourself.

## What is actually on the screen right now

I am publishing this in Phase 0, the online alpha. I work alone, on weekends, and I have a day job. The bar for Phase 0 is: four classes, five specs, one boss, server-authoritative online co-op. Not "running on a LAN", actual online play. That is the bar, and I will tell you on this devlog when each piece of it lands.

Here is what exists in the repository as of this post:

- **Four class controllers.** Gunner (first-person, raycast gun, recoil, line of sight). Vanguard (third-person souls-like, dodge, parry, directional block, two specs). Blade Dancer (state machine, target-lock, four blade configurations, GCD-driven combos). Arcanotechnicien (channeling, commitment windows, a small kit of VFX-driven abilities).
- **One arena.** CSG geometry, cover boxes, pillars, a flat floor. Functional, ugly, deliberate. The point of the arena is to be readable, not pretty.
- **One enemy.** Walks toward the nearest player, swings a telegraphed melee at 1 second wind-up, fires a telegraphed projectile at the farthest player with a half-second laser warning. Two patterns, on a loop. It is enough to test that the controllers can fail and recover.
- **A Go game server.** Gateway, zone, chat. WebSocket transport, 20Hz tick loop, server-authoritative combat resolution. Hub and arena zones, zone transfer on portal walk. Groups, invites, lobby flow. The game runs on a real network, not on localhost.
- **Flux and Resistance systems** server-side. Reserve, afflux, recovery, instability on the Flux side; RMEC, RRAD, RINT damage types on the Resistance side. Still rough. Still real.

I am not going to show you a trailer. The trailer is the second post. This one is the receipts.

## What this devlog is

Every week, I will try to publish one concrete thing that works. A new system, a new enemy pattern, a new piece of art (that i clumsly made), a new slice of level. With footage, not mockups. With the actual broken bits still in the actual broken state, because pretending everything works on the first try is how indie devs lie to themselves.

I will not:

- Promise release dates. I cant. I do stuff when i can
- Announce features I have not started.
- Pretend the project is further along than it is.
- Sell you on a vision of the game I cannot yet build.

If that sounds like the kind of project you want to follow, the [subscribe form](/#subscribe) sends you exactly one short email a week. No newsletter, no drip campaign, no upsell.

If you want to help, the project is open source. Any help is welcome. Im not a game dev. I can do the backend all right, but the client art and code is hard for me.

## The first thing I will show you

Next week: the four Phase 0 classes and the boss, in the same arena, online. Four different cameras, four different input models, one shared fight, five players in a hub going through the portal. It is janky. It is real. I think you will like it.

See you next week.
