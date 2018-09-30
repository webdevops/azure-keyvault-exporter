package main

import (
	"fmt"
	"time"
	"math/rand"
)

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


func randomTime(base, randTime time.Duration) time.Duration {
	sleepTime := int(base.Seconds()) + rand.Intn(int(randTime.Seconds()))
	duration, err := time.ParseDuration(fmt.Sprintf("%ds", sleepTime))
	if err != nil {
		panic(err)
	}

	return duration
}
