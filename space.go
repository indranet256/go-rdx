package rdx

type Space Branch

func MakeSpace(handle, title string, misc Stage, key *KeyPair) (sha Sha256, err error) {
	id := ID{key.KeyLet(), 0}
	err = misc.Add(MakeEulerOf(id, []RDX{
		MakeTuple(ID0, MakeTerm("type").AppendTerm("space")),
	}))
	if err != nil {
		return
	}
	sha, err = MakeBranch(handle, title, misc, key)
	return
}
