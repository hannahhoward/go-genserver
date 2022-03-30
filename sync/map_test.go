package sync_test

import (
	"testing"

	"github.com/hannahhoward/go-genserver/sync"
	"github.com/stretchr/testify/require"
)

func TestMap(t *testing.T) {
	testCases := map[string]func(*testing.T, *sync.Map[int, int]){
		"store and load key": func(t *testing.T, testMap *sync.Map[int, int]) {
			val, exists := testMap.Load(5)
			require.Equal(t, 0, val)
			require.False(t, exists)
			testMap.Store(5, 2)
			val, exists = testMap.Load(5)
			require.Equal(t, 2, val)
			require.True(t, exists)
		},
		"store and loadAndDelete key": func(t *testing.T, testMap *sync.Map[int, int]) {
			val, exists := testMap.LoadAndDelete(5)
			require.Equal(t, 0, val)
			require.False(t, exists)
			testMap.Store(5, 2)
			val, exists = testMap.LoadAndDelete(5)
			require.Equal(t, 2, val)
			require.True(t, exists)
			val, exists = testMap.LoadAndDelete(5)
			require.Equal(t, 0, val)
			require.False(t, exists)
		},
		"loadOrStore key": func(t *testing.T, testMap *sync.Map[int, int]) {
			val, loaded := testMap.LoadOrStore(5, 2)
			require.Equal(t, 2, val)
			require.False(t, loaded)
			testMap.Store(5, 2)
			val, loaded = testMap.LoadOrStore(5, 4)
			require.Equal(t, 2, val)
			require.True(t, loaded)
		},
	}
	for testCase, test := range testCases {
		t.Run(testCase, func(t *testing.T) {
			var testMap sync.Map[int, int]
			test(t, &testMap)
		})
	}
}
