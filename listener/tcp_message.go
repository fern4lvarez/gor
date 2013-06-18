package listener

import (
	"sort"
	"time"
)

const MSG_EXPIRE = 200 * time.Millisecond

// TCPMessage ensure that all TCP packets for given request is received, and processed in right sequence
// Its needed because all TCP message can be fragmented or re-transmitted
//
// Each TCP Packet have 2 ids: acknowledgement - message_id, and sequence - packet_id
// Message can be compiled from unique packets with same message_id which sorted by sequence
// Message is received if we did't receive any packets for 200ms OR if we received packet with "fin" flag
type TCPMessage struct {
	Ack     uint32 // Message ID
	packets []*TCPPacket

	timer *time.Timer

	expired bool

	c_packets chan *TCPPacket
	c_closing chan int

	c_del_message chan *TCPMessage
}

func NewTCPMessage(Ack uint32, c_del chan *TCPMessage) (msg *TCPMessage) {
	msg = &TCPMessage{Ack: Ack}

	msg.c_packets = make(chan *TCPPacket)
	msg.c_closing = make(chan int)
	msg.c_del_message = c_del

	msg.timer = time.AfterFunc(MSG_EXPIRE, msg.Timeout)

	go msg.ListenPackets()

	return
}

func (t *TCPMessage) ListenPackets() {
	for {
		select {
		case <-t.c_closing:
			close(t.c_packets)
			return
		case packet := <-t.c_packets:
			t.AddPacket(packet)
		}
	}
}

func (t *TCPMessage) Timeout() {
	t.c_closing <- 1
	t.c_del_message <- t
}

// Sort packets in right orders and return message content
func (t *TCPMessage) Bytes() (output []byte) {
	mk := make([]int, len(t.packets))

	i := 0
	for k, _ := range t.packets {
		mk[i] = k
		i++
	}

	sort.Ints(mk)

	for _, k := range mk {
		output = append(output, t.packets[k].Data...)
	}

	return
}

// Add packet to the message
func (t *TCPMessage) AddPacket(packet *TCPPacket) {
	if t.expired {
		Debug("Adding packet to expired message")
		return
	}

	packetFound := false

	for _, pkt := range t.packets {
		if packet.Seq == pkt.Seq {
			packetFound = true
			break
		}
	}

	if packetFound {
		Debug("Received packet with same sequence")
	} else {
		t.packets = append(t.packets, packet)
	}

	t.timer.Reset(MSG_EXPIRE)
}
