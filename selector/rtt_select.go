package selector

import (
	"github.com/sipt/shuttle"
	"time"
	"sync/atomic"
)

const (
	timerDulation = 10 * time.Minute
)

func init() {
	shuttle.RegisterSelector("rtt", func(group *shuttle.ServerGroup) (shuttle.ISelector, error) {
		s := &rttSelector{
			group:    group,
			timer:    time.NewTimer(timerDulation),
			selected: group.Servers[0].(shuttle.IServer),
		}
		go func() {
			for {
				select {
				case <-s.timer.C:
				}
				s.autoTest()
				s.timer.Reset(timerDulation)
			}
		}()
		go s.Refresh()
		return s, nil
	})
}

type rttSelector struct {
	group    *shuttle.ServerGroup
	selected shuttle.IServer
	status   uint32
	timer    *time.Timer
}

func (m *rttSelector) Get() (*shuttle.Server, error) {
	return m.selected.GetServer()
}
func (m *rttSelector) Select(name string) error {
	return nil
}
func (m *rttSelector) Refresh() error {
	m.autoTest()
	return nil
}
func (m *rttSelector) Reset(group *shuttle.ServerGroup) error {
	m.group = group
	m.selected = m.group.Servers[0].(shuttle.IServer)
	go m.autoTest()
	return nil
}
func (m *rttSelector) autoTest() {
	if m.status == 0 {
		if ok := atomic.CompareAndSwapUint32(&m.status, 0, 1); !ok {
			return
		}
	}
	shuttle.Logger.Debug("[RTT-Selector] start testing ...")
	var is shuttle.IServer
	var s *shuttle.Server
	var err error
	c := make(chan *shuttle.Server, 1)
	for _, v := range m.group.Servers {
		is = v.(shuttle.IServer)
		s, err = is.GetServer()
		if err != nil {
			continue
		}
		go urlTest(s, c)
	}
	s = <-c
	shuttle.Logger.Infof("[RTT-Select] rtt select server: [%s]", s.Name)
	m.selected = s
	m.timer.Reset(timerDulation)
	atomic.CompareAndSwapUint32(&m.status, 1, 0)
}

func urlTest(s *shuttle.Server, c chan *shuttle.Server) {
	var closer func()
	start := time.Now()
	conn, err := s.Conn(shuttle.TCP)
	if err != nil {
		shuttle.Logger.Debugf("[RTT-Select] [%s]  url test result: <failed> %v", s.Name, err)
		return
	}
	defer conn.Close()
	eAddr, err := shuttle.DomainEncodeing("www.gstatic.com:80")
	if err != nil {
		shuttle.Logger.Debugf("[RTT-Select] [%s]  url test result: <failed> %v", s.Name, err)
		return
	}
	_, err = conn.Write(eAddr)
	if err != nil {
		shuttle.Logger.Debugf("[RTT-Select] [%s]  url test result: <failed> %v", s.Name, err)
		return
	}
	_, err = conn.Write([]byte("GET http://www.gstatic.com/generate_204 HTTP/1.1\r\nHost: www.gstatic.com\r\nAccept: */*\r\nProxy-Connection: Keep-Alive\r\n\r\n"))
	if err != nil {
		shuttle.Logger.Debugf("[RTT-Select] [%s]  url test result: <failed> %v", s.Name, err)
		return
	}
	buf := make([]byte, 128)
	_, err = conn.Read(buf)
	if err != nil {
		shuttle.Logger.Debugf("[RTT-Select] [%s]  url test result: <failed> %v", s.Name, err)
		return
	}
	if err == nil && string(buf[9:12]) == "204" {
		select {
		case c <- s:
		default:
		}
	}
	if err != nil {
		shuttle.Logger.Debugf("[RTT-Select] [%s]  url test result: <failed> %v", s.Name, err)
	} else {
		shuttle.Logger.Debugf("[RTT-Select] [%s]  RTT:[%dms]", s.Name, time.Now().Sub(start).Nanoseconds()/1000000)
	}
	if closer != nil {
		closer()
	}
}