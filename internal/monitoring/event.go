package monitoring

import (
	"crypto/md5"
	"fmt"
	"strconv"
	"strings"
	"time"
)

var (
	addEmojiBeforeTextReplacer = strings.NewReplacer(
		"Сбор", "\n⏱️ Сбор",
	)
)

type Event struct {
	Date        string
	Time        string
	Book        string
	Price       string
	Description string
	Remaining   string
}

func (e Event) GetSum() string {
	return fmt.Sprintf("%x", md5.Sum([]byte(e.Date+e.Time+e.Book+e.Description+e.Price)))
}

func (e Event) BuildMessage() string {
	return fmt.Sprintf(
		`📆 *%s %s*

🎤 *%s*
💷 *%s*
🎟️ %s
🎫 [*Купить билеты*](%s)
`, e.Date, e.Time, addEmojiBeforeTextReplacer.Replace(e.Description), e.Price, e.Remaining, e.Book+"?rand="+strconv.FormatInt(time.Now().Unix(), 32),
	)
}
