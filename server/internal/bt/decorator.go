package bt

// Inverter wraps a child and flips Success to Failure and vice versa.
// Running passes through unchanged.
type Inverter struct {
	Child Node
}

func NewInverter(child Node) *Inverter {
	return &Inverter{Child: child}
}

func (inv *Inverter) Tick(ctx any) Result {
	r := inv.Child.Tick(ctx)
	switch r {
	case Success:
		return Failure
	case Failure:
		return Success
	default:
		return Running
	}
}
