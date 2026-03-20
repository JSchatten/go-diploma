package utils

func LuhnCheck(number string) bool {
	// тупо стащил с википедии
	if len(number) == 0 {
		return false
	}

	sum := 0
	parity := len(number) & 1

	for i := 0; i < len(number); i++ {
		digit := int(number[i] - '0')
		if digit < 0 || digit > 9 {
			return false
		}
		if i&1 == parity {
			digit *= 2
			if digit > 9 {
				digit -= 9
			}
		}
		sum += digit
	}

	return sum%10 == 0
}
