package base62

const alphabet = "0123456789ABCDEFGHIJKLMNOPQRSTUVWXYZabcdefghijklmnopqrstuvwxyz"

// Encode 将无符号整数编码为 base62 字符串
func Encode(num uint64) string {
	if num == 0 {
		return alphabet[:1]
	}

	buf := make([]byte, 0, 11)
	for num > 0 {
		buf = append(buf, alphabet[num%62])
		num /= 62
	}

	// 编码结果低位在前, 反转后返回
	for i, j := 0, len(buf)-1; i < j; i, j = i+1, j-1 {
		buf[i], buf[j] = buf[j], buf[i]
	}
	return string(buf)
}
