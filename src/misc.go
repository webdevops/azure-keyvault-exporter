package main

func boolToFloat64(b bool) float64 {
	if b {
		return 1
	}
	return 0
}

func prefixSlice(prefix string, valueMap []string) (ret []string) {
	for _, value := range valueMap {
		ret = append(ret, prefix + value)
	}
	return
}
