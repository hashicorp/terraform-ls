package addrs

type ProviderReferences map[LocalProviderConfig]Provider

func (pr ProviderReferences) LocalNameByAddr(addr Provider) (LocalProviderConfig, bool) {
	for lName, pAddr := range pr {
		if pAddr.Equals(addr) {
			return lName, true
		}
	}
	return LocalProviderConfig{}, false
}
