// 6 classes. Mirrors docs/design/classes/README.md. Keep in sync.
//
// Accent colors are deliberately muted (per docs/design/ui-language.md:
// "restrained color usage"). One controlled accent per class.
//
// Locale-neutral fields (slug, accent, playable) live on CLASS_BASE; the
// translatable name/genre/identity live in CLASS_TEXT keyed by locale. Use
// getClasses(lang) to get the merged, ordered roster.

import { defaultLang, type Lang } from "../i18n/utils";

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

interface ClassBase {
	slug: string
	accent: ClassAccent
	playable: boolean
}

const CLASS_BASE: ClassBase[] = [
	{ slug: 'gunner', accent: 'gunner', playable: true },
	{ slug: 'vanguard', accent: 'vanguard', playable: true },
	{ slug: 'arcanotechnicien', accent: 'arcanotechnicien', playable: true },
	{ slug: 'engineer', accent: 'engineer', playable: false },
	{ slug: 'blade-dancer', accent: 'blade-dancer', playable: true },
	{ slug: 'tutelaire', accent: 'tutelaire', playable: false },
]

interface ClassText {
	name: string
	genre: string
	identity: string
}

const CLASS_TEXT: Record<Lang, Record<string, ClassText>> = {
	en: {
		gunner: {
			name: 'Gunner',
			genre: 'First-Person Shooter',
			identity: 'Lives and dies by your aim and your magazine, always on the move. Assault, marksman, and area denial.',
		},
		vanguard: {
			name: 'Vanguard',
			genre: 'Souls-like Action Melee',
			identity: 'Reads the boss and waits for the opening to punish. A swirling blade, a space-holding shield, or a flanking shadow.',
		},
		arcanotechnicien: {
			name: 'Arcanotechnicien',
			genre: 'Tactical Channeling',
			identity: 'Stops moving to commit to long channels that hit hard. Destroyer nukes, a weaving battlemage, or a healing zone.',
		},
		engineer: {
			name: 'Engineer',
			genre: 'Deployable Management',
			identity: 'Fights through whatever you have put down on the field. Turrets, a piloted drone, or disruption fields.',
		},
		'blade-dancer': {
			name: 'Blade Dancer',
			genre: 'Positional Combo Fighter',
			identity: 'Every ability flips your blades into a new shape, so you are always mid-combo. Two blades for burst, more for sustained AoE.',
		},
		tutelaire: {
			name: 'Tutelaire',
			genre: 'Aura Positioning',
			identity: 'Picks a spot and holds it while the aura does the work. Tanking, retribution, or healing.',
		},
	},
	fr: {
		gunner: {
			name: 'Tireur',
			genre: 'Jeu de tir à la première personne',
			identity: "Tout repose sur votre visée et votre chargeur, toujours en mouvement. Assaut, tir de précision et interdiction de zone.",
		},
		vanguard: {
			name: 'Avant-garde',
			genre: 'Action mêlée façon Souls',
			identity: "Lit le boss et attend l'ouverture pour punir. Une lame tourbillonnante, un bouclier qui tient l'espace, ou une ombre qui contourne.",
		},
		arcanotechnicien: {
			name: 'Arcanotechnicien',
			genre: 'Incantation tactique',
			identity: "S'arrête de bouger pour s'engager dans de longues incantations qui frappent fort. Bombes du destructeur, mage de bataille mobile, ou zone de soin.",
		},
		engineer: {
			name: 'Ingénieur',
			genre: 'Gestion de déploiements',
			identity: "Combat à travers tout ce que vous avez posé sur le terrain. Tourelles, drone piloté, ou champs de perturbation.",
		},
		'blade-dancer': {
			name: 'Danseur de lames',
			genre: 'Combattant à combos positionnels',
			identity: "Chaque capacité réagence vos lames dans une nouvelle forme, vous êtes donc toujours en plein combo. Deux lames pour la salve, davantage pour l'AoE soutenue.",
		},
		tutelaire: {
			name: 'Tutelaire',
			genre: "Positionnement d'aura",
			identity: "Choisit un emplacement et le tient pendant que l'aura fait le travail. Tank, rétribution, ou soin.",
		},
	},
}

/** The full roster, ordered, with text resolved for `lang`. */
export function getClasses(lang: Lang): ClassEntry[] {
	const text = CLASS_TEXT[lang] ?? CLASS_TEXT[defaultLang]
	return CLASS_BASE.map((base) => ({
		...base,
		...text[base.slug],
	}))
}
