package main

//******【Gold Ten：金十动力策略】*******
//1.请手工保证帐号上的钱够！
//2.本策略还不支持单帐号多实例等复杂场景。

//数据流程：
//1 每天收盘后5点整理datafeed，包含1天的tick 5天的1分 30天的5分 所有的日线；计算出5天的3分 30天的15分的数据；tick数据由datarecorder提供，用于做回放模拟交易；分钟和日线数据由第三方提供，用于初始化策略
//2 开盘前，运行ctpgateway datafeed datarecorder，收集tick；用于策略补tick和回放模拟交易
//3 开盘前/后，运行策略，先补各周期120根bar初始化dataframe，然后补当天的tick继续初始化dataframe。注意：盘中中断后，策略重新运行逻辑和盘前运行都是一样的。
//4 ontick驱动dataframe，开始基于dataframe驱动状态策略和交易策略

import (
	"log"
	"time"
)
import . "github.com/sunwangme/bfgo/bftraderclient"
import . "github.com/sunwangme/bfgo/api/bfgateway"
import . "github.com/sunwangme/bfgo/api/bfdatafeed"
import "github.com/oneywang/bfgo/qite"

var trader *Trader
var dataframes qite.DataFrames

//======
type BfClient struct {
	*BfTrderClient
	clientId     string
	tickHandler  bool
	tradeHandler bool
	logHandler   bool
	symbol       string
	exchange     string
}

//======
func (client *BfClient) OnStart() {
	log.Printf("OnStart")
	// 初始化
	trader = NewTrader(client)
	dataframes = make(map[BfBarPeriod]*qite.BarSeries)

	// 获取历史bar
	for _, period := range myPeriodKeyList {
		// 基于tick生成Bar，并在得到完整bar时插入db
		t := time.Now().String()
		dataframes[period] = qite.NewBarSeries(period, t)
		log.Printf("load histroy bars")
		bars, err := client.GetBar(&BfGetBarReq{
			Symbol:   client.symbol,
			Exchange: client.exchange,
			Period:   period,
			ToDate:   "20160606",
			ToTime:   "22:35:00",
			Count:    int32(qite.GOLDTEN_MIN_K_NUM - 1)}) //确保本策略启动后至少1分钟后才开始交易
		if err != nil {
			log.Printf("Error loading histroy bars: %v", err)
		}
		log.Printf("bars: %v", len(bars))
		for _, v := range bars {
			dataframes[period].AppendBar(v)
		}
	}
}

func (client *BfClient) OnNotification(resp *BfNotificationData) {
	// 连接上gw，对于一些重要的事件，gw会发通知，便于策略控制逻辑。
	log.Printf("OnNotification")
	log.Printf("%v", resp)
	// OnTradeWillBegin第一个消息
	// 比如：发出获取当前仓位请求	client.QueryPosition()
	// OnGotContracts第二个消息
}
func (client *BfClient) OnPing(resp *BfPingData) {
	log.Printf("OnPing")
	log.Printf("%v", resp)
}
func (client *BfClient) OnTick(tick *BfTickData) {
	log.Printf("OnTick")
	log.Printf("%v", tick)

	// 按策略需要拼（更新）数据
	for i := range myPeriodKeyList {
		dataframes[myPeriodKeyList[i]].AppendTick(tick)
	}
	// 数据拼（更新）好了，剩下的交给策略自己去
	trader.OnTick(&dataframes)
}

func (client *BfClient) OnError(resp *BfErrorData) {
	log.Printf("OnError")
	log.Printf("%v", resp)

}
func (client *BfClient) OnLog(resp *BfLogData) {
	log.Printf("OnLog")
	log.Printf("%v", resp)
}
func (client *BfClient) OnTrade(resp *BfTradeData) {
	// 挂单的成交
	log.Printf("OnTrade")
	log.Printf("%v", resp)

	if resp.Symbol == client.symbol && resp.Exchange != client.exchange {
		trader.OnTrade(resp)
	}
}
func (client *BfClient) OnOrder(resp *BfOrderData) {
	log.Printf("OnOrder")
	log.Printf("%v", resp)
	// 挂单的中间状态，一般只需要在OnTrade里面处理。
}
func (client *BfClient) OnPosition(resp *BfPositionData) {
	log.Printf("OnPosition")
	log.Printf("%v", resp)
	// ?resp不是个数组吗？
}
func (client *BfClient) OnAccount(resp *BfAccountData) {
	log.Printf("OnAccount")
	log.Printf("%v", resp)
}
func (client *BfClient) OnStop() {
	log.Printf("OnStop, cancle all pending orders")
	// TODO: 退出前，把挂单都撤了
}

//======
func main() {
	client := &BfClient{
		BfTrderClient: NewBfTraderClient(),
		clientId:      "DualCross",
		tickHandler:   true,
		tradeHandler:  false,
		logHandler:    false,
		symbol:        "rb1610",
		exchange:      "SHFE"}

	qite.CurrentFramework.SetClient(client.BfTrderClient)

	BfRun(client,
		client.clientId,
		client.tickHandler,
		client.tradeHandler,
		client.logHandler,
		client.symbol,
		client.exchange)
}
