package lang

func parseLabels(blockType string, schema LabelSchema, parsed []string) []*ParsedLabel {
	labels := make([]*ParsedLabel, len(schema))

	for i, l := range schema {
		var value string
		if len(parsed)-1 >= i {
			value = parsed[i]
		}
		labels[i] = &ParsedLabel{
			Name:  l.Name,
			Value: value,
		}
	}

	return labels
}
