package monitoring

import (
	"fmt"
	"net/http"
	"os"
	"strings"
	"time"

	"github.com/PuerkitoBio/goquery"
	"go.uber.org/zap"
)

const (
	standUpStoreURL = "https://standupstore.ru/"
)

var (
	monthNameReplacer = strings.NewReplacer(
		"Январь", "января",
		"Февраль", "февраля",
		"Март", "марта",
		"Апрель", "апреля",
		"Май", "мая",
		"Июнь", "июня",
		"Июль", "июля",
		"Август", "августа",
		"Сентябрь", "сентября",
		"Октябрь", "октября",
		"Ноябрь", "ноября",
		"Декабрь", "декабря",
	)

	escapeReplacer = strings.NewReplacer(
		"(", "\\(",
		")", "\\)",
		"+", "\\+",
		"-", "\\-",
		"  ", " ",
	)
)

type Watcher struct {
	httpClient http.Client
	logger     *zap.Logger
	interval   time.Duration
	cache      map[string]struct{}
}

func NewWatcher(logger *zap.Logger, interval time.Duration) Watcher {
	return Watcher{
		httpClient: http.Client{Timeout: 10 * time.Second},
		logger:     logger,
		interval:   interval,
		cache:      map[string]struct{}{},
	}
}

func (w *Watcher) enrichWithMeta(event *Event) {
	resp, err := w.httpClient.Get(event.Book)
	if err != nil {
		w.logger.Error("unable to get book page", zap.Error(err), zap.String("url", event.Book))
		return
	}
	bookDoc, err := goquery.NewDocumentFromReader(resp.Body)
	if err != nil {
		w.logger.Error("unable to parse book page", zap.Error(err), zap.String("url", event.Book))
		return
	}
	_ = resp.Body.Close()

	if price, found := bookDoc.Find(".price.tx_price_line").Attr("content"); found {
		event.Price = price + "₽"
	} else {
		event.Price = "Бесплатно"
	}

	if description := bookDoc.Find(".eventon_desc_in").Text(); description != "" {
		event.Description = strings.TrimSpace(escapeReplacer.Replace(description))
	} else {
		event.Description = "Описание отсутствует"
	}

	if remaining := bookDoc.Find(".evotx_remaining_stock").Text(); remaining != "" {
		event.Remaining = fmt.Sprintf("*%s*", remaining)
	} else {
		event.Remaining = "Осталось много билетов"
	}

	if cardTime := bookDoc.Find(".evo_eventcard_time_t").Text(); cardTime != "" {
		event.Time = escapeReplacer.Replace(cardTime)
	}
}

func (w *Watcher) streamNewEvents(doc *goquery.Document, stream chan Event) {
	doc.Find(".evo_above_title").Each(func(i int, s *goquery.Selection) {
		if len(s.Children().Nodes) == 0 {
			core := s.Parent().Parent()

			book, _ := core.Parent().Parent().Find("a").First().Attr("href")
			newEvent := Event{
				Date: fmt.Sprintf("%s %s", core.Find(".date").Text(), monthNameReplacer.Replace(strings.Title(core.Find(".month").Text()))),
				Book: book,
			}
			w.enrichWithMeta(&newEvent)

			hashSum := newEvent.GetSum()
			if _, cached := w.cache[hashSum]; !cached {
				w.cache[hashSum] = struct{}{}
				if stream != nil {
					stream <- newEvent
				}
			}
		}
	})
	return
}

func (w *Watcher) Watch(shutdown <-chan os.Signal) chan Event {
	var (
		updatesChan = make(chan Event)
		send        = false
		tick        = time.NewTicker(time.Second)
	)
	go func() {
		for {
			select {
			case <-shutdown:
				w.logger.Info("triggered shutdown...")
				close(updatesChan)
				w.logger.Info("updates channel closed")
				return
			case <-tick.C:
				w.logger.Info("fetching new events", zap.String("url", standUpStoreURL))
				resp, err := w.httpClient.Get(standUpStoreURL)
				if err != nil {
					w.logger.Error("unable to make http request", zap.Error(err), zap.String("url", standUpStoreURL))
					continue
				}

				doc, err := goquery.NewDocumentFromReader(resp.Body)
				_ = resp.Body.Close()
				if err != nil {
					w.logger.Error("unable to read body", zap.Error(err), zap.String("url", standUpStoreURL))
					continue
				}

				if send {
					w.streamNewEvents(doc, updatesChan)
				} else {
					w.streamNewEvents(doc, nil)
					send = true
				}
				w.logger.Info("finished new events streaming iteration", zap.String("url", standUpStoreURL))
				tick = time.NewTicker(w.interval)
			}
		}
	}()
	return updatesChan
}
