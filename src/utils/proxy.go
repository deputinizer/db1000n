package utils

import (
	"fmt"
	"math/rand"
	"net"
	"net/url"
	"strings"
	"time"

	"github.com/valyala/fasthttp/fasthttpproxy"
	"golang.org/x/net/proxy"
	"h12.io/socks"
)

type ProxyFunc func(network, addr string) (net.Conn, error)

type ProxyParams struct {
	// URLs separated with comma (,)
	// Not a proxy chain. List of proxies to choose from
	URLs string
	// Local IPv4 or IPv6 to dial from
	LocalAddr net.Addr
	// Name of network device / interface to dial from
	Interface string
	// net.Dialer timeout
	Timeout time.Duration
	// True to use HTTP proxy
	HTTPEnabled bool
}

func GetProxyFunc(params ProxyParams) ProxyFunc {
	direct := &net.Dialer{Timeout: params.Timeout, LocalAddr: params.LocalAddr, Control: BindToInterface(params.Interface)}
	if params.URLs == "" {
		return proxy.FromEnvironmentUsing(direct).Dial
	}

	// Not a proxy chain. List of proxies to choose from
	proxies := strings.Split(params.URLs, ",")

	// We need to dial new proxy on each call
	return func(network, addr string) (net.Conn, error) {
		u, err := url.Parse(proxies[rand.Intn(len(proxies))]) //nolint:gosec // Cryptographically secure random not required
		if err != nil {
			return nil, fmt.Errorf("error building proxy %v: %w", u.String(), err)
		}

		switch u.Scheme {
		case "socks5", "socks5h":
			client, err := proxy.FromURL(u, direct)
			if err != nil {
				return nil, fmt.Errorf("error building proxy %v: %w", u.String(), err)
			}

			return client.Dial(network, addr)
		case "socks4", "socks4a":
			return socks.Dial(u.String())(network, addr)
		default:
			if params.HTTPEnabled {
				return fasthttpproxy.FasthttpHTTPDialerTimeout(u.Host, params.Timeout)(addr)
			}

			return nil, fmt.Errorf("unsupported proxy scheme %v", u.Scheme)
		}
	}
}

func ResolveAddr(network, addr string) net.Addr {
	if addr == "" {
		return nil
	}

	var zone string

	// handle ipv6 zone
	if strings.Contains(addr, "%") {
		split := strings.Split(addr, "%")
		addr, zone = split[0], split[1]
	}

	ip := net.ParseIP(addr)

	switch network {
	case "tcp", "tcp4", "tcp6":
		return &net.TCPAddr{IP: ip, Zone: zone}
	case "udp", "udp4", "udp6":
		return &net.UDPAddr{IP: ip, Zone: zone}
	default:
		return &net.IPAddr{IP: ip}
	}
}
