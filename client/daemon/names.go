package daemon

import (
	"fmt"
	"strconv"
	"strings"
)

func ValidateName(name string) error {
	quoted := strconv.Quote(name)
	if quoted[1:len(quoted)-1] != name || strings.ContainsAny(name, "/\\") {
		return fmt.Errorf("name %s contains invalid characters", quoted)
	}
	return nil
}
