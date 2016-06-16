package main

//******TBLR*******
//1.请手工保证帐号上的钱够！
//2.本策略还不支持单帐号多实例等复杂场景。
//3.策略退出时会清除所有挂单。

//********A.TBLR三状态*********
//1.状态就是计算当前价格属于上涨还是下跌、震荡。
//2.每个周期（日，小时，15分钟，3分钟）都有各自的状态。
//3.三状态具体计算算法：
//	上涨：60均线上，MACD红（线上红）
//	震荡：60均线下，MACD红（线下红）/60均线上，MACD绿（线上绿）
//	下跌：60均线下，MACD绿（线下绿）

//********B.TBLR九宫格*********
//	 慢	上涨   震荡   下跌
// 快
//上涨	多	  多	     空
//震荡	多	  -		 空
//下跌	多	  空	     空
//用法：
//1、快慢两个周期的三态生成九宫格，八格都有明确的交易方向
//2、目的是回避慢周期上大级别交易做反，实现震荡和上涨都可以轻松赚钱，扛一扛就可以赚了
//3、周线级别太大，而且可以体现在日线中；周线级别的上涨与下跌可以重仓参与，周线的意义更加在于发现大牛市/大熊市，然后加大杠杆大赚一把
//4、用两个级别，小时+日，两个都满足的时段，才可以交易（小时、日级别的交易做反，都可能爆仓），具体说就是日线和小时线的上涨、震荡状态才可以做多，下跌坚决不做多。
//5、用两个交易策略；一个策略只做多；一个策略只做空。两个策略的可交易时段是不同的，但双震荡过渡期有重叠
//6、波长和波幅除了交易，也可以在这里加以利用，比如日线/小时绿的前10根不做多，上一波的80%波长波幅内不操作
//7、非理性/非教条，想干就干，先拿起再放下，拿不起放不下；记得带套

//********C.TBLR交易策略*********
//“交易策略”由“状态构成的九宫格”决定。操作周期选择：
//60分钟震荡就是3分钟做（买卖都在3分钟），60分上涨就是15分做（买在3分钟，卖在15分钟）
//1.买卖操作：
//	买点：斜率向下背离红，斜率走平出金叉（30/60），斜率向上踩线红
//	加仓点（可全仓操作）：回踩红（最推荐）
//	止盈：三线止盈（破15/30/60均线，各清仓1/3）+死叉止盈+波长止盈+波幅止盈
//	止损：前低
//
//2.操作逻辑：
//	3分钟的震荡入场，赌震荡转上涨下跌，赌成功了就换成15分钟操作，
//	就是背离红开仓，踩线红加仓，然后：1)若大涨，60分钟明显突破，就赌赢了，用15分钟3线止盈；2)如果60分钟没什么改变，就在3分钟上3线止盈。
//	级别上是在3分钟是买或者卖，或者在15分钟是卖，15分钟上没有买
//	遇到大牛市了，3分钟频繁操作不好，得换大级别
//
//3.仓位：
//震荡：1/3仓位
//上涨：左侧：1/3；右侧：2/3

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
	// 发出获取当前仓位请求
	client.QueryPosition()

}
func (client *DualCross) OnTradeWillBegin(resp *BfNotificationData) {
	// 盘前启动策略，能收到这个消息，而且是第一个消息
	// TODO：这里是做初始化的一个时机
	log.Printf("OnTradeWillBegin")
	log.Printf("%v", resp)
}

func (client *DualCross) OnGotContracts(resp *BfNotificationData) {
	// 盘前启动策略，能收到这个消息，是第二个消息
	// TODO：这里是做初始化的一个时机
	log.Printf("OnGotContracts")
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
