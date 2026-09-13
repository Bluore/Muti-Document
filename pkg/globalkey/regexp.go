package globalkey

import "regexp"

var (
	RegexpEmail = regexp.MustCompile("^[\\w-]+@[\\w-]+.[a-zA-Z]{2,4}$")
)
