package main

import (
	"log"
	"math"
	"strconv"
	"strings"
)
import . "github.com/sunwangme/bfgo/api/bfgateway"
import . "github.com/sunwangme/bfgo/api/bfdatafeed"

var periodKeyList = []BfBarPeriod{
	BfBarPeriod_PERIOD_M03,
	BfBarPeriod_PERIOD_M05,
	BfBarPeriod_PERIOD_M10,
	BfBarPeriod_PERIOD_M15,
	BfBarPeriod_PERIOD_M30}

var periodMinutesList = map[BfBarPeriod]int32{
	BfBarPeriod_PERIOD_M03: 3,
	BfBarPeriod_PERIOD_M05: 5,
	BfBarPeriod_PERIOD_M10: 10,
	BfBarPeriod_PERIOD_M15: 15,
	BfBarPeriod_PERIOD_M30: 30}

const (
	BFTICKTIMELAYOUT string = "2006010215:04:05.000"
	BFBARTIMELAYOUT  string = "2006010215:04:00"
)

// "%H:%M:%S.%f"==>"%H:%M:%S"
func Ticktime2Bartime(t string) string {
	var b string
	//	if tt, err := time.Parse(t, BFTICKTIMELAYOUT); err != nil {
	//		b = tt.Format(BFBARTIMELAYOUT)
	//		}

	if dot := strings.LastIndex(t, "."); dot >= 0 {
		b = t[:dot]
	} else {
		log.Fatalf("Failed ticktime2bartime : %s", t)
	}
	return b
}

func Bartime2Minute(t string) int32 {
	var m int32
	if strings.Count(t, ":") != 2 {
		log.Fatalf("Failed bartime2minute : %s", t)
	}
	start := strings.Index(t, ":")
	stop := strings.LastIndex(t, ":")
	if stop > start {
		i, err := strconv.Atoi(t[start+1 : stop])
		if err != nil {
			log.Fatalf("Failed bartime2minute : %s, %v", t, err)
		} else {
			m = int32(i)
		}
	}
	return m
}

// 用Tick数据赋值Bar
func Tick2Bar(t *BfTickData, period BfBarPeriod, b *BfBarData) {
	b.Symbol = t.Symbol
	b.Exchange = t.Exchange
	b.Period = period

	b.ActionDate = t.ActionDate
	b.BarTime = Ticktime2Bartime(t.TickTime) //TODO: "%H:%M:%S.%f"==>"%H:%M:%S"
	b.Volume = t.Volume
	b.OpenInterest = t.OpenInterest
	b.LastVolume = t.LastVolume

	b.OpenPrice = t.LastPrice
	b.HighPrice = t.LastPrice
	b.LowPrice = t.LastPrice
	b.ClosePrice = t.LastPrice
}

type BarSlice map[BfBarPeriod]*BfBarData

type Bars struct {
	// 不同品种当前的1分钟K线
	data map[string]*BarSlice

	// 要把contract保存到datafeed里面才会看到数据
	// 判断是否初始化了这个保存动作的标志
	contractInited bool
}

func (p *Bars) GetBar(id string, period BfBarPeriod) *BfBarData {
	//log.Printf("%v", p.data)
	if bar, ok := p.data[id]; ok {
		return (*bar)[period]
	}
	return nil

}

func (p *Bars) SetBar(id string, bar *BfBarData, period BfBarPeriod) {
	if b, ok := p.data[id]; ok {
		(*b)[period] = bar
	} else {
		// 这个品种第一次赋值
		var bs BarSlice = make(map[BfBarPeriod]*BfBarData)
		bs[period] = bar
		p.data[id] = &bs
	}
}

func Barxxtime2Minute(t string, period BfBarPeriod) int32 {
	if x, ok := periodMinutesList[period]; ok {
		return Bartime2Minute(t) / x * x
	} else {
		panic("Bartime2Minute")
	}
}

func (p *Bars) M01ToMxx(id string, bar *BfBarData, period BfBarPeriod) (*BfBarData, bool) {
	var ret = BfBarData{}
	var newBar = false

	d, ok := p.data[id]
	if !ok {
		panic("imposible")
	}

	if barxx, ok := (*d)[period]; !ok {
		// 这个周期的bar第一次赋值
		ret = *bar
		ret.Period = period
		(*d)[period] = &ret
	} else {
		// 判断是否能够组成一个完整的bar了
		currentMinute := Barxxtime2Minute(bar.BarTime, period)
		previousMinute := Barxxtime2Minute(barxx.BarTime, period)
		if currentMinute == previousMinute {
			// 还在同一个周期中，更新即可
			barxx.Volume = bar.Volume
			barxx.OpenInterest = bar.OpenInterest
			barxx.LastVolume += bar.LastVolume
			barxx.HighPrice = math.Max(bar.HighPrice, barxx.HighPrice)
			barxx.LowPrice = math.Min(bar.LowPrice, barxx.LowPrice)
			barxx.ClosePrice = bar.ClosePrice
		} else {
			// 新的周期开始，需要插入了
			newBar = true
			ret = *barxx
			// 用1分钟的bar初始化一个新的barxx
			*barxx = *bar //TODO：希望这是深拷贝
			barxx.Period = period
		}
	}

	return &ret, newBar
}
