package iqfeed

import (
	//"strings"
	"time"
)

// RegionalMsg A regional update message. See complete message definition in Regional Messages. (http://www.iqfeed.net/dev/api/docs/RegionalMessageFormat.cfm).
type LevelII struct {
// 	Raw           []string 
	Raw           []byte 

}

// UnMarshall sends the data into the usable struct for consumption by the application.
func (r *LevelII) UnMarshall(d []byte, loc *time.Location) {
// 	items := strings.Split(string(d), ",")
	copy(r.Raw, d []byte)
	
	//r.Symbol = items[0]
	//r.Exchange = items[1]
	//r.RegBid = GetFloatFromStr(items[2])
	//r.RegBidSize = GetIntFromStr(items[3])
	//r.RegBidTime = GetTimeInHMS(items[4], loc)
	//r.RegAsk = GetFloatFromStr(items[5])
	//r.RegAskSize = GetIntFromStr(items[6])
	//r.RegAskTime = GetTimeInHMS(items[7], loc)
	//r.FractionDispCode = GetIntFromStr(items[8])
	//r.DecPrecision = GetIntFromStr(items[9])
	//r.MarketCenter = GetIntFromStr(items[10])
}
