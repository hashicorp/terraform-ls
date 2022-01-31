package job

type ID string

func (id ID) String() string {
	return string(id)
}

type IDs []ID

func (ids IDs) Copy() IDs {
	newIds := make([]ID, len(ids))

	for i, id := range ids {
		newIds[i] = id
	}

	return newIds
}
