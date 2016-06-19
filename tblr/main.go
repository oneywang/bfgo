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
	"time"
)
import . "github.com/sunwangme/bfgo/bftraderclient"
import . "github.com/sunwangme/bfgo/api/bfgateway"
import . "github.com/sunwangme/bfgo/api/bfdatafeed"
import "github.com/sunwangme/bfgo/oneywang/bar"

var converter *bar.Converter = bar.NewConverter()
var dataframes DataFrames

//======
type TradeClient struct {
	*BfTrderClient
	clientId     string
	tickHandler  bool
	tradeHandler bool
	logHandler   bool
	symbol       string
	exchange     string
}

func buy(client *TradeClient, price float64, volume int32) string {
	log.Printf("%v", time.Now())
	log.Printf("To Buy: price=%10.3f vol=%d", price, volume)
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
	}

	return resp.BfOrderId
}

func sell(client *TradeClient, price float64, volume int32) string {
	log.Printf("%v", time.Now())
	log.Printf("To sell: price=%10.3f vol=%d", price, volume)
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
	}

	return resp.BfOrderId
}

func short(client *TradeClient, price float64, volume int32) string {
	log.Printf("%v", time.Now())
	log.Printf("short: price=%10.3f vol=%d", price, volume)
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
	}
	return resp.BfOrderId
}

func cover(client *TradeClient, price float64, volume int32) string {
	log.Printf("%v", time.Now())
	log.Printf("To cover: price=%10.3f vol=%d", price, volume)
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
	}
	return resp.BfOrderId
}

//======
func (client *TradeClient) OnStart() {
	log.Printf("OnStart")
	// 发出获取当前仓位请求
	client.QueryPosition()
	// 获取历史bar
	for i := range bar.PeriodKeyList {
		// 基于tick生成Bar，并在得到完整bar时插入db
		period := bar.PeriodKeyList[i]
		t := time.Now().String()
		dataframes[period] = newDataframe(period, t)
		log.Printf("load histroy bars")
		bars, err := client.GetBar(&BfGetBarReq{
			Symbol:   client.symbol,
			Exchange: client.exchange,
			Period:   period,
			ToDate:   "",
			ToTime:   "",
			Count:    int32(SLOW_K_NUM - 1)}) //确保本策略启动后至少1分钟后才开始交易
		if err != nil {
			for i := range bars {
				dataframes[period].appendBar(bars[i])
			}
		}
	}

}
func (client *TradeClient) OnTradeWillBegin(resp *BfNotificationData) {
	// 盘前启动策略，能收到这个消息，而且是第一个消息
	// TODO：这里是做初始化的一个时机
	log.Printf("OnTradeWillBegin")
	log.Printf("%v", resp)
}

func (client *TradeClient) OnGotContracts(resp *BfNotificationData) {
	// 盘前启动策略，能收到这个消息，是第二个消息
	// TODO：这里是做初始化的一个时机
	log.Printf("OnGotContracts")
	log.Printf("%v", resp)
}
func (client *TradeClient) OnPing(resp *BfPingData) {
	log.Printf("OnPing")
	log.Printf("%v", resp)
}
func (client *TradeClient) OnTick(tick *BfTickData) {
	//log.Printf("OnTick")
	//log.Printf("%v", tick)
	for i := range bar.PeriodKeyList {
		// 基于tick生成Bar，并在得到完整bar时插入db
		period := bar.PeriodKeyList[i]
		bar, isNew := converter.Tick2Bar(tick, period)
		log.Printf("Insert %v bar [%s]", period, tick.TickTime)
		log.Printf("%v", bar)
		if isNew {
			// TODO: 买还是卖？
		}
	}

}

func (client *TradeClient) OnError(resp *BfErrorData) {
	log.Printf("OnError")
	log.Printf("%v", resp)

}
func (client *TradeClient) OnLog(resp *BfLogData) {
	log.Printf("OnLog")
	log.Printf("%v", resp)
}
func (client *TradeClient) OnTrade(resp *BfTradeData) {
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
func (client *TradeClient) OnOrder(resp *BfOrderData) {
	log.Printf("OnOrder")
	log.Printf("%v", resp)
	// 挂单的中间状态，一般只需要在OnTrade里面处理。
}
func (client *TradeClient) OnPosition(resp *BfPositionData) {
	log.Printf("OnPosition")
	log.Printf("%v", resp)
	// ?resp不是个数组吗？
	if resp.Symbol == client.symbol && resp.Exchange == client.exchange {
		initPosition(resp)
	}
}
func (client *TradeClient) OnAccount(resp *BfAccountData) {
	log.Printf("OnAccount")
	log.Printf("%v", resp)
}
func (client *TradeClient) OnStop() {
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
	client := &TradeClient{
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
