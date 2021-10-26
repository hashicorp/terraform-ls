package state

type ModuleChangeHook func(oldMod, newMod *Module)

type ModuleChangeHooks []ModuleChangeHook

func (mh ModuleChangeHooks) notifyModuleChange(oldMod, newMod *Module) {
	for _, h := range mh {
		h(oldMod, newMod)
	}
}
