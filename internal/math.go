package internal

import "math/big"

func GCD(periods []int64) int64 {
	if len(periods) == 0 {
		return 0
	}

	b64 := big.NewInt(periods[0])
	for _, p := range periods[1:] {
		b64.GCD(nil, nil, b64, big.NewInt(p))
	}

	return b64.Int64()
}
