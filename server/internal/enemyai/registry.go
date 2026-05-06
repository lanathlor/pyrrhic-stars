package enemyai

// DefRegistry maps def names to their definitions. Populated at startup
// via LoadMobs() (Tier 1/2) and LoadEncounters() (Tier 3 bosses).
var DefRegistry = map[string]*EnemyDef{}
