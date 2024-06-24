package util

import (
	"github.com/pkg/errors"
	log "github.com/sirupsen/logrus"
	"regexp"
	"strconv"
	"time"
)

// StringToInt16 converts a string to a int16 returning a default value on failure
// noinspection GoUnusedExportedFunction
func StringToInt16(s string, def int16) int16 {
	v, err := strconv.ParseInt(s, 10, 16)
	if err != nil {
		log.Warnf("failed to parse int16 value from redis: %s", s)
		return def
	}
	return int16(v)
}

// StringToUInt16 converts a string to a uint16 returning a default value on failure
func StringToUInt16(s string, def uint16) uint16 {
	v, err := strconv.ParseInt(s, 10, 32)
	if err != nil {
		log.Warnf("failed to parse uint16 value: %s", s)
		return def
	}
	return uint16(v)
}

// StringToUInt converts a string to a uint32 returning a default value on failure
func StringToUInt(s string, def int) int {
	v, err := strconv.ParseInt(s, 10, 32)
	if err != nil {
		log.Warnf("failed to parse uint32 value: %s", s)
		return def
	}
	return int(v)
}

// StringToInt32 converts a string to a uint32 returning a default value on failure
func StringToInt32(s string, def int32) int32 {
	v, err := strconv.ParseInt(s, 10, 32)
	if err != nil {
		log.Warnf("failed to parse uint32 value: %s", s)
		return def
	}
	return int32(v)
}

// StringToUInt32 converts a string to a uint32 returning a default value on failure
func StringToUInt32(s string, def uint32) uint32 {
	v, err := strconv.ParseInt(s, 10, 32)
	if err != nil {
		log.Warnf("failed to parse uint32 value: %s", s)
		return def
	}
	return uint32(v)
}

// StringToUInt64 converts a string to a uint32 returning a default value on failure
func StringToUInt64(s string, def uint64) uint64 {
	v, err := strconv.ParseInt(s, 10, 64)
	if err != nil {
		log.Warnf("failed to parse uint64 value: %s", s)
		return def
	}
	return uint64(v)
}

// StringToFloat64 converts a string to a float64 returning a default value on failure
func StringToFloat64(s string, def float64) float64 {
	v, err := strconv.ParseFloat(s, 64)
	if err != nil {
		log.Warnf("failed to parse float64 value: %s", s)
		return def
	}
	return v
}

// StringToBool converts a string to a bool returning a default value on failure
func StringToBool(s string, def bool) bool {
	v, err := strconv.ParseBool(s)
	if err != nil {
		log.Warnf("failed to parse bool value: %s", s)
		return def
	}
	return v
}

// StringToTime converts a string to a time.Time returning a default value on failure
func StringToTime(s string) time.Time {
	v, err := time.Parse(time.RFC1123Z, s)
	if err != nil {
		log.Warnf("failed to parse time value: %s", s)
		return time.Now()
	}
	return v
}

// TimeToString converts a time.Time to a common RFC1123Z string
func TimeToString(t time.Time) string {
	return t.Format(time.RFC1123Z)
}

var (
	reDuration *regexp.Regexp

	errInvalidDuration = errors.New("Invalid duration")
)

// ParseDuration works exactly like time.ParseDuration except that
// it supports durations longer than hours
// Formats: s, m, h, d, w, M, y
func ParseDuration(s string) (time.Duration, error) {
	if s == "0" {
		return 0, nil
	}
	m := reDuration.FindStringSubmatch(s)
	if m == nil {
		return 0, errInvalidDuration
	}
	valueInt, err := strconv.ParseInt(m[1], 10, 64)
	if err != nil {
		return 0, errInvalidDuration
	}
	value := time.Duration(valueInt)
	day := time.Hour * 24
	switch m[2] {
	case "s":
		return time.Second * value, nil
	case "m":
		return time.Minute * value, nil
	case "h":
		return time.Hour * value, nil
	case "d":
		return day * value, nil
	case "w":
		return day * 7 * value, nil
	case "M":
		return day * 31 * value, nil
	case "y":
		return day * 365 * value, nil
	}
	return 0, errInvalidDuration
}

func init() {
	reDuration = regexp.MustCompile(`^(\d+)([smhdwMy])$`)
}
