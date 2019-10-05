package watertower

import (
	"fmt"
	"strings"
)

type CombinedError struct {
	Message string
	Errors  []error
}

func (c *CombinedError) append(err error) {
	c.Errors = append(c.Errors, err)
}

func (c *CombinedError) appendIfError(err error) {
	if err != nil {
		c.append(err)
	}
}

func (c CombinedError) Error() string {
	var result []string
	for _, err := range c.Errors {
		result = append(result, err.Error())
	}
	return fmt.Sprintf("%s: %s", c.Message, strings.Join(result, ", "))
}
