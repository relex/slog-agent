package tparsetime

func atoi2(s string) int {
	v := int((s[0]-'0'))*10 +
		int((s[1] - '0'))
	return v
}

func atoi4(s string) int {
	v := int((s[0]-'0'))*1000 +
		int((s[1]-'0'))*100 +
		int((s[2]-'0'))*10 +
		int((s[3] - '0'))
	return v
}

func atof3(s string) float64 {
	v := float64((s[1]-'0'))*0.1 +
		float64((s[2]-'0'))*0.01 +
		float64((s[3]-'0'))*0.001
	return v
}

func atof6(s string) float64 {
	v := float64((s[1]-'0'))*0.1 +
		float64((s[2]-'0'))*0.01 +
		float64((s[3]-'0'))*0.001 +
		float64((s[4]-'0'))*0.0001 +
		float64((s[5]-'0'))*0.00001 +
		float64((s[6]-'0'))*0.000001
	return v
}

func atof9(s string) float64 {
	v := float64((s[1]-'0'))*0.1 +
		float64((s[2]-'0'))*0.01 +
		float64((s[3]-'0'))*0.001 +
		float64((s[4]-'0'))*0.0001 +
		float64((s[5]-'0'))*0.00001 +
		float64((s[6]-'0'))*0.000001 +
		float64((s[7]-'0'))*0.0000001 +
		float64((s[8]-'0'))*0.00000001 +
		float64((s[9]-'0'))*0.000000001
	return v
}
