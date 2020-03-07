package bot

import (
	"log"
	"strings"
)

const backtickReplyFormatter = twoEscapeReplyFormatter("`")

type replyFormatter interface {
	Format(string) string
}

type twoEscapeReplyFormatter string

func (f twoEscapeReplyFormatter) Format(message string) string {
	l := string(f)
	if strings.Contains(message, l+l) {
		log.Printf("The message to print has the escape character: %q", message) // TODO: warn log?
	}

	return l + l + message + l + l
}
