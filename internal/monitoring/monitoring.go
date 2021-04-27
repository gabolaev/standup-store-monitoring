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

func (w *Watcher) findNewInStockEvents(doc *goquery.Document) []Event {
	events := make([]Event, 0, len(w.cache))

	doc.Find(".evo_above_title").Each(func(i int, s *goquery.Selection) {
		if len(s.Children().Nodes) == 0 {
			core := s.Parent().Parent()

			book, _ := core.Parent().Parent().Find("a").First().Attr("href")
			newEvent := Event{
				Date: fmt.Sprintf("%s %s", core.Find(".date").Text(), strings.Title(core.Find(".month").Text())),
				Time: core.Find(".evcal_desc2.evcal_event_title").Text(),
				Book: book,
			}

			hashSum := newEvent.GetSum()
			if _, cached := w.cache[hashSum]; !cached {
				w.cache[hashSum] = struct{}{}
				events = append(events, newEvent)
			}
		}
	})
	if len(events) > 0 {
		w.logger.Debug("found new events", zap.Int("count", len(events)))
	} else {
		w.logger.Debug("no new events")
	}
	return events
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

				for _, newEvent := range w.findNewInStockEvents(doc) {
					if send {
						updatesChan <- newEvent
					}
				}
				send = true
				tick = time.NewTicker(w.interval)
			}
		}
	}()
	return updatesChan
}
