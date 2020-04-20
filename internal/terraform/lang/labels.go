package lang

func parseLabels(blockType string, schema LabelSchema, parsed []string) []*Label {
	labels := make([]*Label, len(schema))

	for i, labelName := range schema {
		var value string
		if len(parsed)-1 >= i {
			value = parsed[i]
		}
		labels[i] = &Label{
			Name:  labelName,
			Value: value,
		}
	}

	return labels
}
