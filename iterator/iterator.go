package iterator

import "github.com/ipfans/authgate/proxy"

// Iterator is the iterator pattern implementation created to iterate over proxies
type Iterator interface {
	// Next returns the next proxy to be used. It returns an error if all the proxies
	// turned out to be unavailable
	Next() (*proxy.Proxy, error)
}
