package rdx

type Space Branch

func MakeSpace(handle, title string, misc Stage, key *KeyPair) (sha Sha256, err error) {
	id := ID{key.KeyLet(), 0}
	err = misc.Add(E(id,
		P0(T0("type"), T0("space")),
	))
	if err != nil {
		return
	}
	sha, err = MakeBranch(handle, title, misc, key)
	return
}
