package diameter

import (
	"fmt"
	"net"
	"strings"
	"time"

	"github.com/ishidawataru/sctp"
)

func normalizeTransport(transport string) string {
	t := strings.ToLower(strings.TrimSpace(transport))
	if t == "" {
		return "tcp"
	}
	return t
}

// DialConn dials a Diameter transport endpoint.
func DialConn(transport, addr string, timeout time.Duration) (net.Conn, error) {
	switch normalizeTransport(transport) {
	case "tcp":
		return net.DialTimeout("tcp", addr, timeout)
	case "sctp":
		raddr, err := sctp.ResolveSCTPAddr("sctp", addr)
		if err != nil {
			return nil, fmt.Errorf("resolve SCTP addr %q: %w", addr, err)
		}
		laddr, err := selectSCTPLocalAddr(addr, timeout)
		if err != nil {
			return nil, err
		}

		type result struct {
			conn net.Conn
			err  error
		}
		ch := make(chan result, 1)
		go func() {
			conn, err := sctp.DialSCTP("sctp", laddr, raddr)
			ch <- result{conn: conn, err: err}
		}()

		select {
		case res := <-ch:
			return res.conn, res.err
		case <-time.After(timeout):
			return nil, fmt.Errorf("dial SCTP %s: timeout after %s", addr, timeout)
		}
	default:
		return nil, fmt.Errorf("unsupported diameter transport %q", transport)
	}
}

func selectSCTPLocalAddr(remoteAddr string, timeout time.Duration) (*sctp.SCTPAddr, error) {
	host, _, err := net.SplitHostPort(remoteAddr)
	if err != nil {
		return nil, fmt.Errorf("split SCTP remote addr %q: %w", remoteAddr, err)
	}
	ip := net.ParseIP(host)
	network := "udp4"
	if ip == nil || ip.To4() == nil {
		network = "udp6"
	}
	udpConn, err := net.DialTimeout(network, net.JoinHostPort(host, "9"), timeout)
	if err != nil {
		return nil, fmt.Errorf("select SCTP local addr for %q: %w", remoteAddr, err)
	}
	defer udpConn.Close()

	localUDPAddr, ok := udpConn.LocalAddr().(*net.UDPAddr)
	if !ok || localUDPAddr.IP == nil {
		return nil, fmt.Errorf("resolve local SCTP addr for %q: no local IP", remoteAddr)
	}
	return &sctp.SCTPAddr{
		IPAddrs: []net.IPAddr{{IP: localUDPAddr.IP}},
	}, nil
}

// Listen starts a Diameter listener for the configured transport.
func Listen(transport, addr string) (net.Listener, error) {
	switch normalizeTransport(transport) {
	case "tcp":
		return net.Listen("tcp", addr)
	case "sctp":
		laddr, err := sctp.ResolveSCTPAddr("sctp", addr)
		if err != nil {
			return nil, fmt.Errorf("resolve SCTP listen addr %q: %w", addr, err)
		}
		return sctp.ListenSCTP("sctp", laddr)
	default:
		return nil, fmt.Errorf("unsupported diameter transport %q", transport)
	}
}
