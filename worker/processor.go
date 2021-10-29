package worker

import (
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"
	"unicode"

	"github.com/DataDog/hyperloglog"
	"github.com/giolekva/unique/rpc"
	"golang.org/x/net/html"
)

type StatsConsumer func(rpc.WorkerStatistics, *hyperloglog.HyperLogLog, []string)

type docStats struct {
	hll      *hyperloglog.HyperLogLog
	outbound []string
}

type Processor struct {
	numWorkers   int
	urls         chan []string
	stats        chan *docStats
	numProcessed int
	hll          *hyperloglog.HyperLogLog
	outbound     []string
}

func NewProcessor(numWorkers int, numBits uint) (*Processor, error) {
	hll, err := hyperloglog.New(numBits)
	if err != nil {
		return nil, err
	}
	return &Processor{
		numWorkers:   numWorkers,
		urls:         make(chan []string, 100),
		stats:        make(chan *docStats, 100),
		numProcessed: 0,
		hll:          hll,
		outbound:     make([]string, 0),
	}, nil
}

func (p *Processor) Start(consumer StatsConsumer) {
	for i := 0; i < p.numWorkers; i++ {
		go func() {
			for addresses := range p.urls {
				for _, url := range addresses {
					fmt.Println(url)
					stats, err := processURL(url, p.hll.M)
					if err != nil {
						fmt.Println(err.Error())
						continue
					}
					p.update(stats)
				}
			}
		}()
	}
	for {
		minPeriod := time.After(1 * time.Second)
		select {
		case <-minPeriod:
			p.callConsumer(consumer)
		case stats := <-p.stats:
			if err := p.hll.Merge(stats.hll); err != nil {
				fmt.Println(err.Error())
				break
			}
			p.numProcessed++
			p.outbound = append(p.outbound, stats.outbound...)
			if len(p.outbound) > 10 {
				p.callConsumer(consumer)
			}
		}
	}
}

func (p *Processor) Add(url string) {
	p.urls <- []string{url}
}

func (p *Processor) update(stats *docStats) {
	p.stats <- stats
}

func (p *Processor) callConsumer(consumer StatsConsumer) {
	if p.hll.Count() == 0 {
		return
	}
	consumer(rpc.WorkerStatistics{
		Processed: p.numProcessed,
	}, p.hll, p.outbound)
	p.numProcessed = 0
	if hll, err := hyperloglog.New(p.hll.M); err != nil {
		panic(err)
	} else {
		p.hll = hll
	}
	p.outbound = make([]string, 0)
}

func processURL(address string, numBits uint) (*docStats, error) {
	base, err := url.Parse(address)
	if err != nil {
		return nil, err
	}
	resp, err := http.Get(address)
	if err != nil {
		return nil, err
	}
	return processDocument(resp.Body, base, numBits)
}

func isSeparator(c rune) bool {
	return !unicode.IsLetter(c) && !unicode.IsNumber(c)
}

func processDocument(r io.Reader, base *url.URL, numBits uint) (*docStats, error) {
	doc, err := html.Parse(r)
	if err != nil {
		return nil, err
	}
	hll, err := hyperloglog.New(numBits)
	if err != nil {
		return nil, err
	}
	outbound := make([]string, 0)
	var f func(*html.Node)
	f = func(n *html.Node) {
		if n.Type == html.TextNode {
			for _, w := range strings.FieldsFunc(n.Data, isSeparator) {
				hll.Add(hyperloglog.MurmurString(w))
			}
		} else if n.Type == html.ElementNode && n.Data == "a" {
			for _, a := range n.Attr {
				if a.Key == "href" {
					link, err := url.Parse(a.Val)
					if err != nil {
						continue
					}
					outbound = append(outbound, base.ResolveReference(link).String())
					break
				}
			}
		}
		for c := n.FirstChild; c != nil; c = c.NextSibling {
			f(c)
		}
	}
	f(doc)
	return &docStats{hll, outbound}, nil
}
