package main

//##############Readme#########################
//1.从gw读取TICK合并成分钟BAR存入datafeed！
//2.支持多品种，支持*全品种。
//3.支持多周期：1分钟，3，5，10，15，30分钟，小时，日。
//4.尚不支持的周期：周，月，年。
//5.尚不支持数据有效性、完整性检验--待回测gw可以喂历史数据后做。

import (
	"log"
)
import . "github.com/sunwangme/bfgo/bftraderclient"
import . "github.com/sunwangme/bfgo/api/bfgateway"

//======
type DataRecorder struct {
	*BfTrderClient
	clientId     string
	tickHandler  bool
	tradeHandler bool
	logHandler   bool
	symbol       string
	exchange     string
	bars         *Bars
}

func insertContracts(client *DataRecorder) {
	resps, _ := client.GwGetContract(&BfGetContractReq{Symbol: "*", Exchange: "*"})
	for _, resp := range resps {
		client.InsertContract(resp)
	}
}

//======
func (client *DataRecorder) OnStart() {
	log.Printf("OnStart")
	// 策略每次连接上gw会收到，是做初始化的一个时机。
	//
	// save contracts
	//
	client.bars.contractInited = true
	insertContracts(client)
}
func (client *DataRecorder) OnNotification(resp *BfNotificationData) {
	// 连接上gw，对于一些重要的事件，gw会发通知，便于策略控制逻辑。
	log.Printf("OnNotification")
	log.Printf("%v", resp)
}
func (client *DataRecorder) OnPing(resp *BfPingData) {
	return
	log.Printf("OnPing")
	log.Printf("%v", resp)
}
func (client *DataRecorder) OnTick(tick *BfTickData) {
	//log.Printf("OnTick")
	//log.Printf("%v", tick)

	//
	// save tick
	//
	client.InsertTick(tick)

	// 要把contract保存到datafeed里面才会看到数据
	// ongotcontracts只有ctpgateway连接上ctp时候才发，所有盘中策略连接ctpgateway时候，是没有这个信息的。
	// 可以手工把ctpgateway ctp-stop然后ctp-start以下，就可以得到这个消息。我们这里程序自动判断如果没有调用则主动调用一次。
	if client.bars.contractInited == false {
		client.bars.contractInited = true
		insertContracts(client)
	}
	// 计算K线
	id := tick.Symbol + "@" + tick.Exchange
	// tickDatetime = datetime.strptime(tick.actionDate+tick.tickTime,"%Y%m%d%H:%M:%S.%f")

	if bar, needInsert := client.bars.UpdatM01(id, tick); needInsert {
		log.Printf("Insert 1min bar [%s]", tick.TickTime)
		log.Printf("%v", bar)
		client.InsertBar(bar)

		// 基于M01生成其他Min的Bar
		for i := range periodKeyList {
			if ret, needInsert := client.bars.UpdateMxHDW(id, bar, periodKeyList[i]); needInsert {
				log.Printf("Insert %v bar [%s]", periodKeyList[i], tick.TickTime)
				log.Printf("%v", ret)
				for j := range ret {
					client.InsertBar(ret[j])
				}

			}
		}
	}

}

func (client *DataRecorder) OnError(resp *BfErrorData) {
	log.Printf("OnError")
	log.Printf("%v", resp)

}
func (client *DataRecorder) OnLog(resp *BfLogData) {
	log.Printf("OnLog")
	log.Printf("%v", resp)
}
func (client *DataRecorder) OnTrade(resp *BfTradeData) {
	log.Printf("OnTrade")
	log.Printf("%v", resp)
}
func (client *DataRecorder) OnOrder(resp *BfOrderData) {
	log.Printf("OnOrder")
	log.Printf("%v", resp)
}
func (client *DataRecorder) OnPosition(resp *BfPositionData) {
	log.Printf("OnPosition")
	log.Printf("%v", resp)
}
func (client *DataRecorder) OnAccount(resp *BfAccountData) {
	log.Printf("OnAccount")
	log.Printf("%v", resp)
}
func (client *DataRecorder) OnStop() {
	log.Printf("OnStop")
}

//======
func main() {

	client := &DataRecorder{
		BfTrderClient: NewBfTraderClient(),
		clientId:      "Barsaver",
		tickHandler:   true,
		tradeHandler:  false,
		logHandler:    false,
		symbol:        "rb1610",                                                       //rb1610",
		exchange:      "SHFE",                                                         //"SHFE",
		bars:          &Bars{data: make(map[string]*BarSlice), contractInited: false}} //SHFE"}

	BfRun(client,
		client.clientId,
		client.tickHandler,
		client.tradeHandler,
		client.logHandler,
		client.symbol,
		client.exchange)
}
