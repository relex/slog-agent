// Package syslogprotocol provides shared functions and constants of the syslog RFC 5424 protocol
package syslogprotocol

import (
	"github.com/relex/slog-agent/base"
)

// RFC5424Schema is a sample schema containing all syslog fields in their original form (not from SyslogParser)
// Non-standard field name mapping:
//   - pri => facility and level (severity), both have string values defined below
//   - appname => app
//   - msgid => source
//   - message => log
var RFC5424Schema = base.MustNewLogSchema([]string{"facility", "level", "time", "host", "app", "pid", "source", "extradata", "log"})

// FacilityNames contains the mapping of facility numbers to readable names
var FacilityNames = []string{
	"kern",     // 0
	"user",     // 1
	"mail",     // 2
	"daemon",   // 3
	"auth",     // 4
	"syslog",   // 5
	"lpr",      // 6
	"news",     // 7
	"uucp",     // 8
	"cron",     // 9
	"authpriv", // 10
	"ftp",      // 11
	"ntp",      // 12
	"audit",    // 13
	"alert",    // 14
	"clock",    // 15
	"local0",   // 16
	"local1",   // 17
	"local2",   // 18
	"local3",   // 19
	"local4",   // 20
	"local5",   // 21
	"local6",   // 22
	"local7",   // 23
}

// SeverityNames contains the mapping of severity (level) numbers to readable names
var SeverityNames = []string{
	"emerg",  // 0
	"alert",  // 1
	"crit",   // 2
	"err",    // 3
	"warn",   // 4
	"notice", // 5
	"info",   // 6
	"debug",  // 7
}

// SeverityToLog4jLevel contains the mapping of severity (level) numbers to readable names
var SeverityToLog4jLevel = []string{
	"off",    // 0
	"fatal",  // 1
	"crit",   // 2: not in log4j
	"error",  // 3
	"warn",   // 4
	"notice", // 5: not in log4j
	"info",   // 6
	"debug",  // 7
}
