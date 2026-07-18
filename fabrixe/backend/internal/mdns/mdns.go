// Package mdns advertises the Fabrixe service on the local network.
// Uses only the Go standard library — raw UDP multicast to the mDNS group.
// For production use, avahi-daemon (installed by the install script) handles
// .local resolution independently and more robustly.
package mdns

import (
	"encoding/binary"
	"fmt"
	"log"
	"net"
	"sync"
)

const (
	mdnsGroupIPv4 = "224.0.0.251"
	mdnsPort      = 5353
)

// Server holds the mDNS UDP connection.
type Server struct {
	conn *net.UDPConn
	mu   sync.Mutex
}

// Advertise announces the fabrixe.local service via multicast DNS.
// Returns a Server that can be shut down. Non-fatal on error (logs warning).
func Advertise(hostname string, port int) (*Server, error) {
	addr, err := net.ResolveUDPAddr("udp4", fmt.Sprintf("%s:%d", mdnsGroupIPv4, mdnsPort))
	if err != nil {
		log.Printf("[mdns] resolve error: %v — mDNS unavailable (avahi-daemon handles .local resolution)", err)
		return &Server{}, nil
	}

	conn, err := net.DialUDP("udp4", nil, addr)
	if err != nil {
		log.Printf("[mdns] dial error: %v — mDNS unavailable (avahi-daemon handles .local resolution)", err)
		return &Server{}, nil
	}

	s := &Server{conn: conn}

	// Announce once on startup
	pkt := buildPTRResponse(hostname, port)
	if _, err := conn.Write(pkt); err != nil {
		log.Printf("[mdns] send error: %v", err)
	} else {
		log.Printf("[mdns] Announced %s.local on the LAN (port %d)", hostname, port)
	}

	return s, nil
}

// Shutdown stops the mDNS server.
func (s *Server) Shutdown() {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.conn != nil {
		_ = s.conn.Close()
		s.conn = nil
	}
}

// buildPTRResponse builds a minimal DNS response packet announcing <hostname>.local.
// This is a PTR record for _fabrixe._tcp.local pointing to <hostname>.local.
func buildPTRResponse(hostname string, port int) []byte {
	buf := make([]byte, 0, 256)

	// DNS Header
	// ID = 0, QR = 1 (response), OPCODE = 0, AA = 1 (authoritative), rest = 0
	// Flags: 0x8400 = response + authoritative
	header := []uint16{
		0x0000, // ID
		0x8400, // Flags
		0x0000, // QDCOUNT
		0x0001, // ANCOUNT
		0x0000, // NSCOUNT
		0x0000, // ARCOUNT
	}
	for _, h := range header {
		buf = append(buf, byte(h>>8), byte(h))
	}

	// Answer: PTR record for _fabrixe._tcp.local → <hostname>.local
	// Name: _fabrixe._tcp.local
	buf = appendDNSLabel(buf, "_fabrixe")
	buf = appendDNSLabel(buf, "_tcp")
	buf = appendDNSLabel(buf, "local")
	buf = append(buf, 0x00) // end of name

	// Type PTR (12), Class IN (1), TTL 120, RDLENGTH, RDATA
	buf = append(buf, binary.BigEndian.AppendUint16(nil, 12)...)  // TYPE PTR
	buf = append(buf, binary.BigEndian.AppendUint16(nil, 1)...)   // CLASS IN
	buf = append(buf, binary.BigEndian.AppendUint32(nil, 120)...) // TTL

	// RDATA = <hostname>.local
	rdBuf := make([]byte, 0, 64)
	rdBuf = appendDNSLabel(rdBuf, hostname)
	rdBuf = appendDNSLabel(rdBuf, "local")
	rdBuf = append(rdBuf, 0x00)

	buf = append(buf, binary.BigEndian.AppendUint16(nil, uint16(len(rdBuf)))...)
	buf = append(buf, rdBuf...)

	_ = port // could add SRV record; omitted for simplicity
	return buf
}

func appendDNSLabel(buf []byte, label string) []byte {
	buf = append(buf, byte(len(label)))
	buf = append(buf, []byte(label)...)
	return buf
}
