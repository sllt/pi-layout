package sid

import (
	"math"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
)

func decodeBase62(t *testing.T, s string) uint64 {
	t.Helper()
	require.NotEmpty(t, s)
	var v uint64
	for i := 0; i < len(s); i++ {
		idx := strings.IndexByte(base62, s[i])
		require.GreaterOrEqual(t, idx, 0, "unexpected base62 char %q in %q", s[i], s)
		v = v*62 + uint64(idx)
	}
	return v
}

func TestUint64ToBase62(t *testing.T) {
	require.Equal(t, "0", Uint64ToBase62(0))
	require.Equal(t, "Z", Uint64ToBase62(61))
	require.Equal(t, "10", Uint64ToBase62(62))

	for _, n := range []uint64{0, 1, 61, 62, 123456789, math.MaxInt64, math.MaxUint64} {
		encoded := Uint64ToBase62(n)
		require.Equal(t, n, decodeBase62(t, encoded))
	}
}

func TestIntToBase62(t *testing.T) {
	require.Equal(t, "0", IntToBase62(0))
	require.Equal(t, "0", IntToBase62(-1))
	require.Equal(t, "0", IntToBase62(math.MinInt))
	require.Equal(t, "10", IntToBase62(62))
	require.Equal(t, Uint64ToBase62(uint64(math.MaxInt)), IntToBase62(math.MaxInt))
}

func TestSid_Gen(t *testing.T) {
	s := NewSid()

	strID, err := s.GenString()
	require.NoError(t, err)
	require.NotEmpty(t, strID)
	require.NotZero(t, decodeBase62(t, strID))

	uID, err := s.GenUint64()
	require.NoError(t, err)
	require.NotZero(t, uID)
}
