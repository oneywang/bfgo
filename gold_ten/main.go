package main

//******【Gold Ten】*******
//1.请手工保证帐号上的钱够！
//2.本策略还不支持单帐号多实例等复杂场景。
//3.策略退出时会清除所有挂单。

//******【基于Order的策略框架】*******
//1.onstart
//   初始化各周期dataframe如dfs
//   补最近60根k到dfs
//2.ontick
//   推tick到dfs
//   捕捉行情任务运行一次，基于order的，order成交后，要明确定义止盈止损条件，形成止盈止损任务
//   OpenTaskRunOnce(dfs)
//   止盈止损任务运行一次，基于order的，而不是基于pos
//   CloseTaskRunOnce(dfs)

//******【基于order的task设计】******
//1. 由于同时有多个task在运行，task都是基于技术指标驱动的，所以封装技术指标处理在一起，各个task用就行了，
//   那个东西就叫dataframe吧
//2. 流程:状态策略产生交易策略，交易策略产生开仓任务，开仓成交完成，产生平仓任务...
//3. statemodel:
//   statemodel-->ontick-->trademodel.start/stop
//4. trademodel:
//   opentask-->ontick/ontrade-->openorder-->closetask(s)-->ontick/ontrade-->closeorder(s)

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
