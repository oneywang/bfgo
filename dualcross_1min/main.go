package main

//1.请手工保证帐号上的钱够！
//2.本策略还不支持单帐号多实例等复杂场景。
//3.策略退出时会清除所有挂单。

import (
	"log"
	"strconv"
	"strings"
	"time"
)
import . "github.com/sunwangme/bfgo/bftraderclient"
import . "github.com/sunwangme/bfgo/api/bfgateway"
import . "github.com/sunwangme/bfgo/api/bfdatafeed"

// 本策略的交易参数常量
const (
	TRADE_VOLUME int32  = 1
	VOLUME_LIMIT int32  = 5
	FAST_K_NUM   uint32 = 15
	SLOW_K_NUM   uint32 = 60
)

// 本策略的变量
var _period BfBarPeriod = BfBarPeriod_PERIOD_M01
var _historyBarsGot bool = false
var _barsCount uint32 = 0
var _currentBarMinute uint32 = 0
var _fastMa, _slowMa []float64
var _fastMa0, _fastMa1, _slowMa0, _slowMa1 float64 = 0, 0, 0, 0
var _positionLong, _positionShort int32 = 0, 0
var _pendingOrderIds []string

//======
type DualCross struct {
	*BfTrderClient
	clientId     string
	tickHandler  bool
	tradeHandler bool
	logHandler   bool
	symbol       string
	exchange     string
}

// "%H:%M:%S.%f"==>"%H:%M:%S"
func ticktime2bartime(t string) string {
	var b string
	if dot := strings.LastIndex(t, "."); dot >= 0 {
		b = t[:dot]
	} else {
		log.Fatalf("Failed ticktime2bartime : %s", t)
	}
	return b
}

func bartime2minute(t string) uint32 {
	var m uint32
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
			m = uint32(i)
		}
	}
	return m
}

// 用Tick数据赋值Bar
func tick2bar(t *BfTickData, period BfBarPeriod, b *BfBarData) {
	b.Symbol = t.Symbol
	b.Exchange = t.Exchange
	b.Period = period

	b.ActionDate = t.ActionDate
	b.BarTime = ticktime2bartime(t.TickTime) //TODO: "%H:%M:%S.%f"==>"%H:%M:%S"
	b.Volume = t.Volume
	b.OpenInterest = t.OpenInterest
	b.LastVolume = t.LastVolume

	b.OpenPrice = t.LastPrice
	b.HighPrice = t.LastPrice
	b.LowPrice = t.LastPrice
	b.ClosePrice = t.LastPrice
}

func initPosition(position *BfPositionData) {
	if _positionLong > 0 || _positionShort > 0 {
		// already inited
		return
	}
	if position.Direction == BfDirection_DIRECTION_LONG {
		_positionLong += position.Position
	} else if position.Direction == BfDirection_DIRECTION_SHORT {
		_positionShort += position.Position
	}
}

func updatePosition(direction BfDirection, offset BfOffset, volume int32) {
	if direction == BfDirection_DIRECTION_LONG && offset == BfOffset_OFFSET_OPEN {
		_positionLong += volume
	} else if direction == BfDirection_DIRECTION_LONG && offset == BfOffset_OFFSET_CLOSE {
		_positionLong -= volume
	} else if direction == BfDirection_DIRECTION_SHORT && offset == BfOffset_OFFSET_OPEN {
		_positionShort += volume
	} else if direction == BfDirection_DIRECTION_SHORT && offset == BfOffset_OFFSET_CLOSE {
		_positionShort -= volume
	}
}

func indexOf(a []string, v string) int {
	for i := range a {
		if a[i] == v {
			return i
		}
	}
	return -1
}

func without(a []string, v string) []string {
	var r []string
	j := 0
	for i := range a {
		if a[i] != v {
			r[j] = a[i]
			j++
		}
	}
	return r
}

func onBar(client *DualCross, closePrice float64) {
	// 计算快慢均线
	if 0 == _fastMa0 {
		_fastMa0 = closePrice
	} else {
		_fastMa1 = _fastMa0
		_fastMa0 = (closePrice + _fastMa0*float64(FAST_K_NUM-1)) / float64(FAST_K_NUM)
	}
	_fastMa = append(_fastMa, _fastMa0)

	if 0 == _slowMa0 {
		_slowMa0 = closePrice
	} else {
		_slowMa1 = _slowMa0
		_slowMa0 = (closePrice + _slowMa0*float64(SLOW_K_NUM-1)) / float64(SLOW_K_NUM)
	}
	_slowMa = append(_slowMa, _slowMa0)

	// 判断是否足够bar--初始化时会去历史，如果历史不够，会积累到至少  SLOW_K_NUM 数量的bar才会交易
	_barsCount += 1
	log.Printf("bar count: %d", _barsCount)
	if _barsCount < SLOW_K_NUM {
		return
	}

	// 判断买卖
	crossOver := _fastMa0 > _slowMa0 && _fastMa1 < _slowMa1  // 金叉上穿
	crossBelow := _fastMa0 < _slowMa0 && _fastMa1 > _slowMa1 // 死叉下穿

	if crossOver {
		// 金叉
		// 1.如果有空头持仓，则先平仓
		if _positionShort > 0 {
			cover(client, closePrice, _positionShort)
		}
		// 2.持仓未到上限，则继续做多
		if _positionLong < VOLUME_LIMIT {
			buy(client, closePrice, TRADE_VOLUME)
		}
	} else if crossBelow {
		// 死叉
		// 1.如果有多头持仓，则先平仓
		if _positionLong > 0 {
			sell(client, closePrice, _positionLong)
		}
		// 2.持仓未到上限，则继续做空
		if _positionShort < VOLUME_LIMIT {
			short(client, closePrice, TRADE_VOLUME)
		}
	}
}

func buy(client *DualCross, price float64, volume int32) {
	log.Printf("%v", time.Now())
	resp, err := client.SendOrder(&BfSendOrderReq{
		Symbol:    client.symbol,
		Exchange:  client.exchange,
		Price:     price,
		Volume:    volume,
		PriceType: BfPriceType_PRICETYPE_LIMITPRICE,
		Direction: BfDirection_DIRECTION_LONG,
		Offset:    BfOffset_OFFSET_OPEN})
	if err != nil {
		log.Fatal("Buy error")
	} else {
		log.Printf("Buy: price=%10.3f vol=%d", price, volume)
		_pendingOrderIds = append(_pendingOrderIds, resp.BfOrderId)
	}
}

func sell(client *DualCross, price float64, volume int32) {
	log.Printf("%v", time.Now())
	resp, err := client.SendOrder(&BfSendOrderReq{
		Symbol:    client.symbol,
		Exchange:  client.exchange,
		Price:     price,
		Volume:    volume,
		PriceType: BfPriceType_PRICETYPE_LIMITPRICE,
		Direction: BfDirection_DIRECTION_LONG,
		Offset:    BfOffset_OFFSET_CLOSETODAY})
	if err != nil {
		log.Fatal("sell error")
	} else {
		log.Printf("sell: price=%10.3f vol=%d", price, volume)
		_pendingOrderIds = append(_pendingOrderIds, resp.BfOrderId)
	}
}

func short(client *DualCross, price float64, volume int32) {
	log.Printf("%v", time.Now())
	resp, err := client.SendOrder(&BfSendOrderReq{
		Symbol:    client.symbol,
		Exchange:  client.exchange,
		Price:     price,
		Volume:    volume,
		PriceType: BfPriceType_PRICETYPE_LIMITPRICE,
		Direction: BfDirection_DIRECTION_SHORT,
		Offset:    BfOffset_OFFSET_OPEN})
	if err != nil {
		log.Fatal("short error")
	} else {
		log.Printf("short: price=%10.3f vol=%d", price, volume)
		_pendingOrderIds = append(_pendingOrderIds, resp.BfOrderId)
	}
}

func cover(client *DualCross, price float64, volume int32) {
	log.Printf("%v", time.Now())
	resp, err := client.SendOrder(&BfSendOrderReq{
		Symbol:    client.symbol,
		Exchange:  client.exchange,
		Price:     price,
		Volume:    volume,
		PriceType: BfPriceType_PRICETYPE_LIMITPRICE,
		Direction: BfDirection_DIRECTION_SHORT,
		Offset:    BfOffset_OFFSET_CLOSETODAY})
	if err != nil {
		log.Fatal("cover error")
	} else {
		log.Printf("cover: price=%10.3f vol=%d", price, volume)
		_pendingOrderIds = append(_pendingOrderIds, resp.BfOrderId)
	}
}

//======
func (client *DualCross) OnStart() {
	log.Printf("OnStart")
	// 策略每次连接上gw会收到，是做初始化的一个时机。

	// 发出获取当前仓位请求
	client.QueryPosition()
}
func (client *DualCross) OnNotification(resp *BfNotificationData) {
	// 连接上gw，对于一些重要的事件，gw会发通知，便于策略控制逻辑。
	log.Printf("OnNotification")
	log.Printf("%v", resp)
}
func (client *DualCross) OnPing(resp *BfPingData) {
	log.Printf("OnPing")
	log.Printf("%v", resp)
}
func (client *DualCross) OnTick(tick *BfTickData) {
	//log.Printf("OnTick")
	//log.Printf("%v", tick)

	tickMinute := bartime2minute(ticktime2bartime(tick.TickTime))
	if !_historyBarsGot {
		log.Printf("load histroy bars")
		_historyBarsGot = true
		bars, err := client.GetBar(&BfGetBarReq{
			Symbol:   client.symbol,
			Exchange: client.exchange,
			Period:   _period,
			ToDate:   tick.ActionDate,
			ToTime:   tick.TickTime,
			Count:    int32(SLOW_K_NUM - 1)}) //确保本策略启动后至少1分钟后才开始交易
		if err == nil {
			for i := range bars {
				onBar(client, bars[i].ClosePrice)
			}
		}
		_currentBarMinute = tickMinute
		return
	}

	if tickMinute != _currentBarMinute {
		// 每一新分钟得到K线
		// 因为只用到了bar.closePrice，所以不必再去datafeed取K线
		onBar(client, tick.LastPrice)
		_currentBarMinute = tickMinute
	}
}

func (client *DualCross) OnError(resp *BfErrorData) {
	log.Printf("OnError")
	log.Printf("%v", resp)

}
func (client *DualCross) OnLog(resp *BfLogData) {
	log.Printf("OnLog")
	log.Printf("%v", resp)
}
func (client *DualCross) OnTrade(resp *BfTradeData) {
	// 挂单的成交
	log.Printf("OnTrade")
	log.Printf("%v", resp)

	if resp.Symbol != client.symbol || resp.Exchange != client.exchange {
		return
	}

	if indexOf(_pendingOrderIds, resp.BfOrderId) == -1 {
		// TODO：不是本策略本次运行发起的交易
		return
	}
	// 按最新成交结果：1.更新orderids, 2.更新当前仓位
	_pendingOrderIds = without(_pendingOrderIds, resp.BfOrderId)
	updatePosition(resp.Direction, resp.Offset, resp.Volume)
}
func (client *DualCross) OnOrder(resp *BfOrderData) {
	log.Printf("OnOrder")
	log.Printf("%v", resp)
	// 挂单的中间状态，一般只需要在OnTrade里面处理。
}
func (client *DualCross) OnPosition(resp *BfPositionData) {
	log.Printf("OnPosition")
	log.Printf("%v", resp)
	// ?resp不是个数组吗？
	if resp.Symbol == client.symbol && resp.Exchange == client.exchange {
		initPosition(resp)
	}
}
func (client *DualCross) OnAccount(resp *BfAccountData) {
	log.Printf("OnAccount")
	log.Printf("%v", resp)
}
func (client *DualCross) OnStop() {
	log.Printf("OnStop, cancle all pending orders")
	// 退出前，把挂单都撤了
	req := &BfCancelOrderReq{Symbol: client.symbol, Exchange: client.exchange}
	for i := range _pendingOrderIds {
		req.BfOrderId = _pendingOrderIds[i]
		client.CancleOrder(req)
	}
}

//======
func main() {
	client := &DualCross{
		BfTrderClient: NewBfTraderClient(),
		clientId:      "DualCross",
		tickHandler:   true,
		tradeHandler:  false,
		logHandler:    false,
		symbol:        "rb1610",
		exchange:      "SHFE"}

	BfRun(client,
		client.clientId,
		client.tickHandler,
		client.tradeHandler,
		client.logHandler,
		client.symbol,
		client.exchange)
}
