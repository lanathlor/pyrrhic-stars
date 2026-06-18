// 6 classes. Mirrors docs/design/classes/README.md. Keep in sync.
//
// Accent colors are deliberately muted (per docs/design/ui-language.md:
// "restrained color usage"). One controlled accent per class.

export interface ClassEntry {
	slug: string
	name: string
	genre: string
	identity: string
	/** Tailwind 4 token name without the `--color-class-` prefix. */
	accent: ClassAccent
	/** Playable in the current build. Engineer and Tutelaire arrive in Phase 3. */
	playable: boolean
}

export type ClassAccent =
	| 'gunner'
	| 'vanguard'
	| 'arcanotechnicien'
	| 'engineer'
	| 'blade-dancer'
	| 'tutelaire'

export const CLASSES: ClassEntry[] = [
	{
		slug: 'gunner',
		name: 'Gunner',
		genre: 'First-Person Shooter',
		identity: 'Lives and dies by your aim and your magazine, always on the move. Assault, marksman, and area denial.',
		accent: 'gunner',
		playable: true,
	},
	{
		slug: 'vanguard',
		name: 'Vanguard',
		genre: 'Souls-like Action Melee',
		identity: 'Reads the boss and waits for the opening to punish. A swirling blade, a space-holding shield, or a flanking shadow.',
		accent: 'vanguard',
		playable: true,
	},
	{
		slug: 'arcanotechnicien',
		name: 'Arcanotechnicien',
		genre: 'Tactical Channeling',
		identity: 'Stops moving to commit to long channels that hit hard. Destroyer nukes, a weaving battlemage, or a healing zone.',
		accent: 'arcanotechnicien',
		playable: true,
	},
	{
		slug: 'engineer',
		name: 'Engineer',
		genre: 'Deployable Management',
		identity: 'Fights through whatever you have put down on the field. Turrets, a piloted drone, or disruption fields.',
		accent: 'engineer',
		playable: false,
	},
	{
		slug: 'blade-dancer',
		name: 'Blade Dancer',
		genre: 'Positional State Machine',
		identity: 'Every ability flips your blades into a new shape, so you are always mid-combo. Two blades for burst, more for sustained AoE.',
		accent: 'blade-dancer',
		playable: true,
	},
	{
		slug: 'tutelaire',
		name: 'Tutelaire',
		genre: 'Aura Positioning',
		identity: 'Picks a spot and holds it while the aura does the work. Tanking, retribution, or healing.',
		accent: 'tutelaire',
		playable: false,
	},
]
