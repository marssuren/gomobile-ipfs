// Code generated by generate_multiplexer.go; DO NOT EDIT.

package logging

import "net"

func NewMultiplexedTracer(tracers ...*Tracer) *Tracer {
	if len(tracers) == 0 {
		return nil
	}
	if len(tracers) == 1 {
		return tracers[0]
	}
	return &Tracer{
		SentPacket: func(dest net.Addr, hdr *Header, size ByteCount, frames []Frame) {
			for _, t := range tracers {
				if t.SentPacket != nil {
					t.SentPacket(dest, hdr, size, frames)
				}
			}
		},
		SentVersionNegotiationPacket: func(dest net.Addr, destConnID ArbitraryLenConnectionID, srcConnID ArbitraryLenConnectionID, versions []Version) {
			for _, t := range tracers {
				if t.SentVersionNegotiationPacket != nil {
					t.SentVersionNegotiationPacket(dest, destConnID, srcConnID, versions)
				}
			}
		},
		DroppedPacket: func(addr net.Addr, packetType PacketType, size ByteCount, reason PacketDropReason) {
			for _, t := range tracers {
				if t.DroppedPacket != nil {
					t.DroppedPacket(addr, packetType, size, reason)
				}
			}
		},
		Debug: func(name string, msg string) {
			for _, t := range tracers {
				if t.Debug != nil {
					t.Debug(name, msg)
				}
			}
		},
		Close: func() {
			for _, t := range tracers {
				if t.Close != nil {
					t.Close()
				}
			}
		},
	}
}
