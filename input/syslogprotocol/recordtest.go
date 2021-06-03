package syslogprotocol

// TestRecordStart checks whether given []byte is possibly a valid syslog record
func TestRecordStart(s []byte) bool {
	// ex: "<3>1 2019-08-15T15:50:46 h i 1 n"
	// ex: "<166>1 2019-08-15T15:50:46 h i 1 n"
	//      01234
	if len(s) < 32 {
		return false
	}
	if s[0] != '<' {
		return false
	}
	if s[1] < '0' || s[1] > '9' {
		return false
	}
	var i int
	for i = 2; i < 4; i++ {
		c := s[i]
		if c == '>' {
			return s[i+1] == '1' && s[i+2] == ' '
		}
		if c < '0' || c > '9' {
			return false
		}
	}
	return s[i] == '>' && s[i+1] == '1' && s[i+2] == ' '
}
