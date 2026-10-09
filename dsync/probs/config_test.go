package probs_test

import (
	"github.com/alextanhongpin/core/dsync/probs"
	"github.com/stretchr/testify/require"
	"testing"
)

func TestNilClients(t *testing.T) {
	_, err := probs.NewBloomFilter(nil)
	require.Error(t, err)
	_, err = probs.NewCuckooFilter(nil)
	require.Error(t, err)
	_, err = probs.NewHyperLogLog(nil)
	require.Error(t, err)
	_, err = probs.NewCountMinSketch(nil)
	require.Error(t, err)
	_, err = probs.NewTDigest(nil)
	require.Error(t, err)
	_, err = probs.NewTopK(nil)
	require.Error(t, err)
}
