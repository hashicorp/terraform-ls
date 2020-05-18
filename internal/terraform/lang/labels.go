package lang

func parseLabels(blockType string, schema LabelSchema, parsed []string) []*ParsedLabel {
	labels := make([]*ParsedLabel, len(schema))

	for i, labelName := range schema {
		var value string
		if len(parsed)-1 >= i {
			value = parsed[i]
		}
		labels[i] = &ParsedLabel{
			Name:  labelName,
			Value: value,
		}
	}

	return labels
}
