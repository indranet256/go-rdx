package rdx

type Branch struct {
	Brix  Brix
	Clock ID
	Stage map[ID][]byte
}

func (b *Branch) Open(id ID, hash Sha256) (err error) {
	return
}

func (b *Branch) Tick() ID {
	b.Clock.Seq = (b.Clock.Seq &^ 63) + 64
	return b.Clock
}

// Adds a record change.
func (b *Branch) Add(delta RDX) (err error) {
	return
}

func (b *Branch) Get(id ID) RDX {
	return nil
}

// New creates a record with the content provided;
// must be one RDX element, preferably PLEX.
func (b *Branch) New(delta RDX) error {
	return nil
}

func (b *Branch) Commit() (err error) {
	return nil
}

func (b *Branch) Compact(newHeight int) (err error) {
	return nil
}

func (b *Branch) Close() error {
	return nil
}
