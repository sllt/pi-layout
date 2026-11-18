package sid

const (
	base62 = "0123456789abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ"
)

func Uint64ToBase62(n uint64) string {
	if n == 0 {
		return string(base62[0])
	}

	var buf [11]byte
	i := len(buf)
	for n > 0 {
		i--
		buf[i] = base62[n%62]
		n /= 62
	}

	return string(buf[i:])
}

func IntToBase62(n int) string {
	if n <= 0 {
		return string(base62[0])
	}
	return Uint64ToBase62(uint64(n))
}
