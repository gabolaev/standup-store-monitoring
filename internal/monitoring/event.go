package monitoring

import (
	"crypto/md5"
	"fmt"
	"strconv"
	"time"
)

type Event struct {
	Date string
	Time string
	Book string
}

func (e Event) GetSum() string {
	return fmt.Sprintf("%x", md5.Sum([]byte(e.Date+e.Time+e.Book)))
}

func (e Event) BuildMessage() string {
	return fmt.Sprintf(
		`%s 
%s 

🎟 [Купить билеты](%s)
`,
		e.Date,
		e.Time,
		e.Book+"?rand="+strconv.FormatInt(time.Now().Unix(), 32),
	)
}
