package entity

// DamageLink connects two allies: damage taken by either is split evenly.
// On expiry, the ally with lower HP is healed for a portion of the HP difference.
type DamageLink struct {
	SourcePeer uint16  // caster who created the link
	PeerA      uint16  // first linked ally
	PeerB      uint16  // second linked ally
	Duration   float32 // remaining seconds
}
