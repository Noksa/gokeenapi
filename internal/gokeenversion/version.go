package gokeenversion

var (
	version   = "undefined"
	buildDate = "undefined"
)

func Version() string {
	return version
}

func BuildDate() string {
	return buildDate
}
