package iqfeed

import (
	"bufio"
	"fmt"
	"io"
	"log"
	"net"
	"os"
	"strings"
	"time"
)

// IQC provides the main struct for the the IQ Client interface into what IQFeed will be sending us.
type IQC struct {
	System       chan *SystemMessage
	LevelII      chan *LevelII
	News         chan *NewsMsg
	Errors       chan *ErrorMsg
	Fundamental  chan *FundamentalMsg
	Regional     chan *RegionalMsg
	Time         chan *TimeMsg
	Updates      chan *UpdSummaryMsg
	TimeZone     string
	TimeLoc      *time.Location
	CreateBackup bool
	BackupFile   string
	Conn         net.Conn
	Quit         chan bool
	DynFields    map[int]string
}

func (c *IQC) connect(cs string) {
	if c.TimeZone == "" {
		c.TimeZone = "America/New_York"
	}
	var err error
	c.TimeLoc, err = time.LoadLocation(c.TimeZone)
	if err != nil {
		// We absolutely need the timezone / location so we must panic if it fails.
		panic(err)
	}
	c.DynFields = make(map[int]string)
	if cs == "" {
		cs = "localhost:5009"
	}
	conn, err := net.Dial("tcp", cs)
	if err != nil {
		log.Fatal("Could not connect to IQFeed")
	}
	c.Conn = conn
}

// ProcessSysMsg handles system messages, field definitions are available here: http://www.iqfeed.net/dev/api/docs/Level1SystemMessage.cfm.
func (c *IQC) processSysMsg(d []byte) {
	s := &SystemMessage{}

	pfx := strings.Split(string(d), ",")
	switch pfx[0] {
	case "UPDATE FIELDNAMES":
		/* We use a map here to preserve the actual order as it's important with marshalling dynamic fields */
		for i := 1; i < len(pfx); i++ {
			c.DynFields[i-1] = pfx[i]
		}
	case "CURRENT UPDATE FIELDNAMES":
		/* We use a map here to preserve the actual order as it's important with marshalling dynamic fields */
		for i := 1; i < len(pfx); i++ {
			c.DynFields[i-1] = pfx[i]
		}
	default:
		s.UnMarshall(d, c.TimeLoc)
		c.System <- s
	}
}

// ProcessSumMsg handles summary messages, field definitions are available here: http://www.iqfeed.net/dev/api/docs/Level1UpdateSummaryMessage.cfm.
func (c *IQC) processSumMsg(d []byte) {
	s := &UpdSummaryMsg{}
	items := strings.Split(string(d), ",")
	s.UnMarshall(items, c.DynFields, c.TimeLoc)
	c.Updates <- s
}

// ProcessUpdMsg handles update messages, field definitions are available here: http://www.iqfeed.net/dev/api/docs/Level1UpdateSummaryMessage.cfm.
func (c *IQC) processUpdMsg(d []byte) {
	u := &UpdSummaryMsg{}
	items := strings.Split(string(d), ",")
	if items[2] == "Not Found" {
		c.process404Msg([]byte(items[0]))
		return
	}
	u.UnMarshall(items, c.DynFields, c.TimeLoc)
	c.Updates <- u
}

// ProcessTimeMsg handles timestamp updates, field definitions are available here: http://www.iqfeed.net/dev/api/docs/TimeMessageFormat.cfm.
func (c *IQC) processTimeMsg(d []byte) {
	t := &TimeMsg{}
	t.UnMarshall(d, c.TimeLoc)

	c.Time <- t
}

// ProcessRegUpdMsg handles regional updates field definitions are available here: http://www.iqfeed.net/dev/api/docs/RegionalMessageFormat.cfm.
func (c *IQC) processRegUpdMsg(d []byte) {
	r := &RegionalMsg{}
	r.UnMarshall(d, c.TimeLoc)
	c.Regional <- r
}

// ProcessFndMsg handles fundamental messages, field descriptions are available here: http://www.iqfeed.net/dev/api/docs/Level1FundamentalMessage.cfm.
func (c *IQC) processFndMsg(d []byte) {
	f := &FundamentalMsg{}
	f.UnMarshall(d, c.TimeLoc)
	c.Fundamental <- f

}

// ProcessNewsMsg handles summary messages, field definitions are available here: http://www.iqfeed.net/dev/api/docs/StreamingNewsMessageFormat.cfm.
func (c *IQC) processNewsMsg(d []byte) {
	n := &NewsMsg{}
	n.UnMarshall(d, c.TimeLoc)
	c.News <- n
}

// Process404Msg handles messages indicating that a symbol was not found.
func (c *IQC) process404Msg(d []byte) {
	e := &ErrorMsg{}
	e.UnMarshall(true, d, 404)
	c.Errors <- e
}

// ProcessErrorMsg handles error messages in the form of error text.
func (c *IQC) processErrorMsg(d []byte) {
	e := &ErrorMsg{}
	e.UnMarshall(false, d, 500)
	c.Errors <- e
}

func (c *IQC) processLvl2Msg(d []byte) {
	lvlII := &LevelII{}
	lvlII.UnMarshall(d, c.TimeLoc)
	c.LevelII <- lvlII
}

// ProcessReceiver is one of the main reciever functions that interprets data received by IQFeed and processes it in sub functions.
func (c *IQC) processReceiver(d []byte) {	
	data := []byte{}
	
	//0x4F = "O" = Market Open, it is a single char we need to add a default catch here.  
	// Also it is prob stalling cuz of this in split since it has no more ,
	if sz := len(d); sz > 1 {
		data = d[2:]
	} else {
		data = d[0:]
	}

	switch d[0] {
	case 0x32: // number 2 = Level 2 Message
		c.processLvl2Msg(data)
	case 0x5A: // cap Z = Level 2 Message (initial?)
		c.processLvl2Msg(data)
	case 0x54: // Start letter is T, indicating Time message.
		c.processTimeMsg(data)
	case 0x4F: // "O" = Market Open (79 dec)
		fmt.Println("Market Open")
	// For some reason we get blocked on Sys messages on level IIport.
// 	case 0x53: // Start letter is S, indicating System message (Unicode representation in integer value).
// 		c.processSysMsg(data)
// 	case 0x50: // Start letter is P, indicating a summary message.
// 		c.processSumMsg(data)
// 	case 0x51: // Start letter is Q, indicating an update message.
// 		c.processUpdMsg(data)
// 	case 0x52: // Start letter is R, indicating regional update message
// 		c.processRegUpdMsg(data)
// 	case 0x46: // Start letter is F, indicating a fundamental message
// 		c.processFndMsg(data)
// 	case 0x4e: // Start letter is N, indicating a news message
// 		c.processNewsMsg(data)
// 	case 0x6E: // Start letter is n, indicating Symbol not found message
// 		c.process404Msg(data)
// 	case 0x45: // Start letter is E, error message
// 		c.processErrorMsg(data)
	default:
		// BugFix: Read what is in buffer so the next message processes correctly.
		raw := string(d)
		sz := len(raw)
		fmt.Println("\nUnknown Message Type: ", raw[0], "\nSize: ", sz, "\nRaw: ", raw)
	}

}

// Read function does as expected and reads data from the network stream.
func (c *IQC) read() {
	r := bufio.NewReader(c.Conn)
	for {
		select {
		case <-c.Quit:
			log.Println("Client quitting")
			c.Conn.Close()
			break
		default:
			line, isPrefix, err := r.ReadLine()
			for err == nil && !isPrefix {
				if c.CreateBackup {
					bld := fmt.Sprintf("%s\r\n", string(line))
					c.writeBackup([]byte(bld))
				}
				c.processReceiver(line)
				line, isPrefix, err = r.ReadLine()
			}
			if isPrefix {
				log.Println("buffer size to small")
				//return Do not return and break the loop
			}
			if err != io.EOF {
				log.Println("Pipe closed exiting...")
				c.Conn.Close()
				os.Exit(0)
			}
		}
	}

}

func (c *IQC) getCallChar(t time.Time) string {
	switch t.Month().String() {
	case "January":
		return "A"
	case "February":
		return "B"
	case "March":
		return "C"
	case "April":
		return "D"
	case "May":
		return "E"
	case "June":
		return "F"
	case "July":
		return "G"
	case "August":
		return "H"
	case "September":
		return "I"
	case "October":
		return "J"
	case "November":
		return "K"
	case "December":
		return "L"
	}
	return "A"
}

func (c *IQC) getPutChar(t time.Time) string {
	switch t.Month().String() {
	case "January":
		return "M"
	case "February":
		return "N"
	case "March":
		return "O"
	case "April":
		return "P"
	case "May":
		return "Q"
	case "June":
		return "R"
	case "July":
		return "S"
	case "August":
		return "T"
	case "September":
		return "U"
	case "October":
		return "V"
	case "November":
		return "W"
	case "December":
		return "X"
	}
	return "M"
}

// Start function will start the concurrent functions to read and write data to the and from the network stream.
func (c *IQC) Start(connectString string) *IQC {
	c.connect(connectString)
	c.System = make(chan *SystemMessage)
	c.LevelII = make(chan *LevelII)
	c.News = make(chan *NewsMsg)
	c.Errors = make(chan *ErrorMsg)
	c.Fundamental = make(chan *FundamentalMsg)
	c.Regional = make(chan *RegionalMsg)
	c.Time = make(chan *TimeMsg)
	c.Updates = make(chan *UpdSummaryMsg)
	go c.read()
// 	c.ReqCurrentUpdateFNames()
// 	c.RequestListedMarkets()
	return c

}
