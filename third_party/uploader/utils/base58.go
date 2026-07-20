package utils

// Base58Encode encodes src using Bitcoin-style base58 alphabet.
// Matches wenshushu verify.js / bs58 (same algorithm as rust/src/util/base58.rs).
func Base58Encode(src []byte) string {
	const alphabet = "123456789ABCDEFGHJKLMNPQRSTUVWXYZabcdefghijkmnopqrstuvwxyz"
	if len(src) == 0 {
		return ""
	}
	zeros := 0
	for zeros < len(src) && src[zeros] == 0 {
		zeros++
	}
	size := (len(src)-zeros)*138/100 + 1
	buf := make([]byte, size)
	length := 0
	for _, b := range src[zeros:] {
		carry := int(b)
		j := 0
		i := size
		for i > 0 && (carry != 0 || j < length) {
			i--
			carry += 256 * int(buf[i])
			buf[i] = byte(carry % 58)
			carry /= 58
			j++
		}
		length = j
	}
	i := size - length
	for i < size && buf[i] == 0 {
		i++
	}
	out := make([]byte, 0, zeros+size-i)
	for z := 0; z < zeros; z++ {
		out = append(out, alphabet[0])
	}
	for i < size {
		out = append(out, alphabet[buf[i]])
		i++
	}
	return string(out)
}
